/*-------------------------------------------------------------------------
*
 * pgEdge Natural Language Agent
*
* Copyright (c) 2025 - 2026, pgEdge, Inc.
* This software is released under The PostgreSQL License
*
*-------------------------------------------------------------------------
*/

package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- libClient wrapper tests (Task 4) ---------------------------------

func TestLibClient_Chat_RoundTrip(t *testing.T) {
	// Fake Anthropic server.
	var gotReqBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReqBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_1","type":"message","role":"assistant",
			"content":[{"type":"text","text":"hi back"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":10,"output_tokens":5}
		}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}

	resp, err := client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if string(gotReqBody) == "" {
		t.Fatal("server received no request body")
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(resp.Content))
	}
	tc, ok := resp.Content[0].(TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", resp.Content[0])
	}
	if tc.Text != "hi back" {
		t.Errorf("text = %q", tc.Text)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("stop_reason = %q", resp.StopReason)
	}
}

func TestLibClient_Chat_InjectsSystemPrompt(t *testing.T) {
	var gotReq struct {
		System []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"system"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	_, _ = client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)

	if len(gotReq.System) == 0 {
		t.Fatal("expected system prompt to be sent")
	}
	if !strings.Contains(gotReq.System[0].Text, "PostgreSQL") {
		t.Errorf("system prompt missing expected text; got %q", gotReq.System[0].Text)
	}
	if strings.Contains(gotReq.System[0].Text, "READ-ONLY mode") {
		t.Error("system prompt unexpectedly contains read-only safety text")
	}
}

func TestLibClient_Chat_AppendsReadOnlyPromptWhenSet(t *testing.T) {
	var gotReq struct {
		System []struct {
			Text string `json:"text"`
		} `json:"system"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	client.SetReadOnlyMode(true)
	_, _ = client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)

	if len(gotReq.System) == 0 || !strings.Contains(gotReq.System[0].Text, "READ-ONLY mode") {
		t.Errorf("expected read-only safety prompt in system text; got %+v", gotReq.System)
	}
}

func TestLibClient_Chat_ReturnsErrorFromUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"bad request"}}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	_, err = client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("expected upstream error to surface, got nil")
	}
}

func TestLibClient_Chat_TranslatesToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"m","type":"message","role":"assistant",
			"content":[
				{"type":"text","text":"calling tool"},
				{"type":"tool_use","id":"tu_1","name":"get_weather","input":{"city":"Paris"}}
			],
			"stop_reason":"tool_use",
			"usage":{"input_tokens":10,"output_tokens":5}
		}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	resp, err := client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "weather?"}}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if len(resp.Content) != 2 {
		t.Fatalf("expected 2 content items, got %d", len(resp.Content))
	}
	tc, ok := resp.Content[0].(TextContent)
	if !ok || tc.Text != "calling tool" {
		t.Errorf("content[0] = %+v", resp.Content[0])
	}
	tu, ok := resp.Content[1].(ToolUse)
	if !ok {
		t.Fatalf("content[1] = %T (want ToolUse)", resp.Content[1])
	}
	if tu.ID != "tu_1" || tu.Name != "get_weather" || tu.Input["city"] != "Paris" {
		t.Errorf("tool = %+v", tu)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("stop_reason = %q", resp.StopReason)
	}
}

func TestLibClient_Chat_PopulatesTokenUsageWhenDebug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"m","type":"message","role":"assistant",
			"content":[{"type":"text","text":"x"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":80}
		}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, true)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	resp, err := client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if resp.TokenUsage == nil {
		t.Fatal("expected TokenUsage when debug=true")
	}
	if resp.TokenUsage.PromptTokens != 100 || resp.TokenUsage.CompletionTokens != 50 {
		t.Errorf("usage = %+v", resp.TokenUsage)
	}
	if resp.TokenUsage.CacheReadTokens != 80 {
		t.Errorf("cache_read = %d", resp.TokenUsage.CacheReadTokens)
	}
	if resp.TokenUsage.Provider != "anthropic" {
		t.Errorf("provider = %q", resp.TokenUsage.Provider)
	}
}

func TestLibClient_Chat_NoTokenUsageWhenNotDebug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","content":[{"type":"text","text":"x"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	resp, err := client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.TokenUsage != nil {
		t.Errorf("expected nil TokenUsage when debug=false, got %+v", resp.TokenUsage)
	}
}

func TestLibClient_ListModels_AnthropicOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-3-5-sonnet","type":"model"},{"id":"claude-3-opus","type":"model"}]}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("models = %v", models)
	}
}

func TestNewOllamaClient_ReturnsErrorOnInvalidBaseURL(t *testing.T) {
	// New signature: returns (LLMClient, error).
	_, err := NewOllamaClient("not a url", "llama3", false)
	if err == nil {
		t.Fatal("expected error for invalid base URL, got nil")
	}
}

func TestLibClient_OpenAI_Chat_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_1","object":"chat.completion","created":1,
			"model":"gpt-4o",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hi from openai"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}
		}`))
	}))
	defer server.Close()

	client, err := NewOpenAIClient("test-key", server.URL, "gpt-4o", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewOpenAIClient: %v", err)
	}
	resp, err := client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(resp.Content))
	}
	tc, ok := resp.Content[0].(TextContent)
	if !ok || tc.Text != "hi from openai" {
		t.Errorf("content[0] = %+v", resp.Content[0])
	}
}

func TestLibClient_Ollama_Chat_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ollama /api/chat returns the single non-stream response form.
		if !strings.HasPrefix(r.URL.Path, "/api/chat") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"llama3","created_at":"2026-05-27T00:00:00Z",
			"message":{"role":"assistant","content":"hi from ollama"},
			"done":true,
			"prompt_eval_count":5,"eval_count":3
		}`))
	}))
	defer server.Close()

	client, err := NewOllamaClient(server.URL, "llama3", false)
	if err != nil {
		t.Fatalf("NewOllamaClient: %v", err)
	}
	resp, err := client.Chat(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(resp.Content))
	}
	tc, ok := resp.Content[0].(TextContent)
	if !ok || tc.Text != "hi from ollama" {
		t.Errorf("content[0] = %+v", resp.Content[0])
	}
}
