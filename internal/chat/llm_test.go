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

	if len(gotReqBody) == 0 {
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

func TestLibClient_Chat_AppliesSystemCachingForAnthropic(t *testing.T) {
	var gotReq struct {
		System []struct {
			Type         string `json:"type"`
			Text         string `json:"text"`
			CacheControl *struct {
				Type string `json:"type"`
			} `json:"cache_control,omitempty"`
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
		t.Fatal("expected at least one system block")
	}
	last := gotReq.System[len(gotReq.System)-1]
	if last.CacheControl == nil {
		t.Fatalf("expected cache_control on the last system block; got %+v", last)
	}
	if last.CacheControl.Type != "ephemeral" {
		t.Errorf("cache_control.type = %q, want ephemeral", last.CacheControl.Type)
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

// TestLibClient_Chat_MultiTurnToolLoop exercises the agentic loop pattern
// that the single-turn tests above never reach: the assistant's first
// response (stored as LLMResponse.Content, a []interface{}) is appended to
// the history and passed back into a second Chat call alongside the tool
// result. Before contentToBlocks gained its []interface{} case, replaying
// that stored assistant turn failed to translate, breaking every
// multi-tool conversation. This test guards that path end to end.
func TestLibClient_Chat_MultiTurnToolLoop(t *testing.T) {
	var secondReqBody []byte
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// Turn 1: the model asks to call a tool.
			_, _ = w.Write([]byte(`{
				"id":"m1","type":"message","role":"assistant",
				"content":[
					{"type":"text","text":"let me check"},
					{"type":"tool_use","id":"tu_1","name":"get_weather","input":{"city":"Paris"}}
				],
				"stop_reason":"tool_use",
				"usage":{"input_tokens":10,"output_tokens":5}
			}`))
			return
		}
		// Turn 2: capture what the client replayed, then give a final answer.
		secondReqBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{
			"id":"m2","type":"message","role":"assistant",
			"content":[{"type":"text","text":"It is 72F and sunny in Paris."}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":20,"output_tokens":8}
		}`))
	}))
	defer server.Close()

	client, err := NewAnthropicClient("test-key", server.URL, "claude-x", 4096, 0.7, false)
	if err != nil {
		t.Fatalf("NewAnthropicClient: %v", err)
	}
	ctx := context.Background()

	// Turn 1: the model returns a tool_use response.
	history := []Message{{Role: "user", Content: "weather in Paris?"}}
	resp1, err := client.Chat(ctx, history, nil)
	if err != nil {
		t.Fatalf("first Chat: %v", err)
	}
	if resp1.StopReason != "tool_use" {
		t.Fatalf("turn 1 stop_reason = %q, want tool_use", resp1.StopReason)
	}

	// Mirror the agentic loop in processQuery: store the assistant turn's
	// raw []interface{} content as history, then append the tool result.
	history = append(history, Message{Role: "assistant", Content: resp1.Content})

	var toolUseID string
	for _, item := range resp1.Content {
		if tu, ok := item.(ToolUse); ok {
			toolUseID = tu.ID
		}
	}
	if toolUseID == "" {
		t.Fatal("turn 1 response contained no ToolUse to replay")
	}
	history = append(history, Message{
		Role: "user",
		Content: []ToolResult{{
			Type:      "tool_result",
			ToolUseID: toolUseID,
			Content:   "72F and sunny",
		}},
	})

	// Turn 2: replay the full history. This is the call that failed before
	// the []interface{} case was added to contentToBlocks.
	resp2, err := client.Chat(ctx, history, nil)
	if err != nil {
		t.Fatalf("second Chat (history replay) failed: %v", err)
	}
	if resp2.StopReason != "end_turn" {
		t.Errorf("turn 2 stop_reason = %q, want end_turn", resp2.StopReason)
	}
	if len(resp2.Content) != 1 {
		t.Fatalf("turn 2 expected 1 content item, got %d", len(resp2.Content))
	}
	if tc, ok := resp2.Content[0].(TextContent); !ok || !strings.Contains(tc.Text, "72F") {
		t.Errorf("turn 2 content[0] = %+v", resp2.Content[0])
	}

	// The replayed request must carry the prior assistant tool_use and the
	// tool_result, proving the stored []interface{} content translated.
	body := string(secondReqBody)
	if !strings.Contains(body, "tu_1") {
		t.Errorf("replayed request missing tool_use id tu_1; body=%s", body)
	}
	if !strings.Contains(body, "tool_result") {
		t.Errorf("replayed request missing tool_result block; body=%s", body)
	}
	if !strings.Contains(body, "72F and sunny") {
		t.Errorf("replayed request missing tool result content; body=%s", body)
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
