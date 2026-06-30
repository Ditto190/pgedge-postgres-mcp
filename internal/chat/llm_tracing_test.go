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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pgedge-postgres-mcp/internal/embedding"
)

func TestTracingRoundTripper_LogsRequestAndResponse(t *testing.T) {
	origLevel := embedding.GetLogLevel()
	embedding.SetLogLevel(embedding.LogLevelDebug)
	defer embedding.SetLogLevel(origLevel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var buf bytes.Buffer
	rt := &tracingRoundTripper{
		provider: "anthropic",
		model:    "claude-x",
		inner:    http.DefaultTransport,
		out:      &buf,
	}
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest("POST", server.URL+"/v1/messages", strings.NewReader(`{"hi":1}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}

	logged := buf.String()
	if !strings.Contains(logged, "anthropic") || !strings.Contains(logged, "claude-x") {
		t.Errorf("log missing provider/model: %s", logged)
	}
	if !strings.Contains(logged, `{"hi":1}`) {
		t.Errorf("log missing request body: %s", logged)
	}
	if !strings.Contains(logged, `{"ok":true}`) {
		t.Errorf("log missing response body: %s", logged)
	}
}

func TestNewTracingHTTPClient_PassesThroughOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := newTracingHTTPClient("anthropic", "claude-x")
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestTracingRoundTripper_NoLogWhenBelowDebug(t *testing.T) {
	origLevel := embedding.GetLogLevel()
	embedding.SetLogLevel(embedding.LogLevelInfo)
	defer embedding.SetLogLevel(origLevel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var buf bytes.Buffer
	rt := &tracingRoundTripper{
		provider: "anthropic",
		model:    "claude-x",
		inner:    http.DefaultTransport,
		out:      &buf,
	}
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest("POST", server.URL+"/v1/messages", strings.NewReader(`{"hi":1}`))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}

	if buf.Len() != 0 {
		t.Errorf("expected no log output below LogLevelDebug, got %q", buf.String())
	}
}
