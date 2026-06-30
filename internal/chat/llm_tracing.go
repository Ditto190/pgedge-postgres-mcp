//-------------------------------------------------------------------------
//
// pgEdge Natural Language Agent
//
// Copyright (c) 2025 - 2026, pgEdge, Inc.
// This software is released under The PostgreSQL License
//
//-------------------------------------------------------------------------

package chat

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"pgedge-postgres-mcp/internal/embedding"
)

// tracingHTTPTimeout caps the wall-clock time of a single upstream LLM
// request through the tracing http.Client. Long enough for slow models
// to finish but short enough that a hung connection eventually fails
// rather than blocking the CLI forever.
const tracingHTTPTimeout = 120 * time.Second

// tracingRoundTripper wraps an inner http.RoundTripper and logs the
// request and response bodies to out when the embedding-package log
// level is Debug or Trace. It is used to recover the request/response
// trace logging behaviour that the old hand-rolled clients did inline.
type tracingRoundTripper struct {
	provider string
	model    string
	inner    http.RoundTripper
	out      io.Writer
}

// RoundTrip logs the outgoing request body and incoming response body
// (when tracing is enabled), then returns the response unchanged.
//
// Bodies are read into memory; this is fine for chat traffic (a few KB
// to a few hundred KB), the same trade-off the old code made.
func (t *tracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if embedding.GetLogLevel() < embedding.LogLevelDebug {
		return t.inner.RoundTrip(req)
	}

	var reqBody []byte
	if req.Body != nil {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("tracing: read request body: %w", err)
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	fmt.Fprintf(t.out, "[LLM] [TRACE] %s/%s %s %s body=%s\n",
		t.provider, t.model, req.Method, req.URL.String(), string(reqBody))

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(t.out, "[LLM] [TRACE] %s/%s transport error: %v\n",
			t.provider, t.model, err)
		return resp, err
	}

	if resp.Body != nil {
		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			fmt.Fprintf(t.out, "[LLM] [TRACE] %s/%s status=%d body-read-error: %v\n",
				t.provider, t.model, resp.StatusCode, readErr)
			resp.Body = io.NopCloser(bytes.NewReader(nil))
		} else {
			fmt.Fprintf(t.out, "[LLM] [TRACE] %s/%s status=%d body=%s\n",
				t.provider, t.model, resp.StatusCode, string(respBody))
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
		}
	}

	return resp, nil
}

// newTracingHTTPClient returns an *http.Client whose Transport logs
// the request/response bodies to stderr at Debug/Trace log levels.
// Pass into llm.Options.HTTPClient when constructing a client whose
// upstream traffic should be traced (typically the CLI in debug mode).
func newTracingHTTPClient(provider, model string) *http.Client {
	return &http.Client{
		Timeout: tracingHTTPTimeout,
		Transport: &tracingRoundTripper{
			provider: provider,
			model:    model,
			inner:    http.DefaultTransport,
			out:      os.Stderr,
		},
	}
}
