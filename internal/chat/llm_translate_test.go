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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"
)

func TestToLibMessages_StringContent(t *testing.T) {
	in := []Message{
		{Role: "user", Content: "hello"},
	}
	got, err := toLibMessages(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: []llm.ContentBlock{{Type: llm.BlockText, Text: "hello"}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestToLibMessages_TypedTextContent(t *testing.T) {
	in := []Message{
		{
			Role: "assistant",
			Content: []TextContent{
				{Type: "text", Text: "result"},
			},
		},
	}
	got, err := toLibMessages(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Content) != 1 {
		t.Fatalf("expected one message with one block, got %+v", got)
	}
	if got[0].Role != llm.RoleAssistant {
		t.Errorf("role = %q, want assistant", got[0].Role)
	}
	if got[0].Content[0].Type != llm.BlockText || got[0].Content[0].Text != "result" {
		t.Errorf("block = %+v", got[0].Content[0])
	}
}

func TestToLibMessages_ToolUseContent(t *testing.T) {
	in := []Message{
		{
			Role: "assistant",
			Content: []ToolUse{
				{Type: "tool_use", ID: "tu_1", Name: "get_weather", Input: map[string]interface{}{"city": "Paris"}},
			},
		},
	}
	got, err := toLibMessages(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Content) != 1 {
		t.Fatalf("expected one message with one block, got %+v", got)
	}
	block := got[0].Content[0]
	if block.Type != llm.BlockToolUse {
		t.Errorf("type = %q, want tool_use", block.Type)
	}
	if block.ToolUse == nil {
		t.Fatal("ToolUse is nil")
	}
	if block.ToolUse.ID != "tu_1" || block.ToolUse.Name != "get_weather" {
		t.Errorf("toolUse = %+v", block.ToolUse)
	}
	var input map[string]interface{}
	if err := json.Unmarshal(block.ToolUse.Input, &input); err != nil {
		t.Fatalf("input unmarshal: %v", err)
	}
	if input["city"] != "Paris" {
		t.Errorf("input = %+v, want city=Paris", input)
	}
}

func TestToLibMessages_ToolResultContent(t *testing.T) {
	in := []Message{
		{
			Role: "user",
			Content: []ToolResult{
				{Type: "tool_result", ToolUseID: "tu_1", Content: "72F sunny", IsError: false},
			},
		},
	}
	got, err := toLibMessages(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Content) != 1 {
		t.Fatalf("expected one message with one block, got %+v", got)
	}
	block := got[0].Content[0]
	if block.Type != llm.BlockToolResult {
		t.Errorf("type = %q, want tool_result", block.Type)
	}
	if block.ToolUseID != "tu_1" || block.Text != "72F sunny" || block.IsError {
		t.Errorf("block = %+v", block)
	}
	if got[0].Role != llm.RoleTool {
		t.Errorf("tool-result message must use RoleTool, got %q", got[0].Role)
	}
}

func TestToLibMessages_InterfaceSliceContent(t *testing.T) {
	// The shape an assistant turn takes once stored back into history:
	// LLMResponse.Content is a []interface{} of TextContent and ToolUse.
	in := []Message{
		{
			Role: "assistant",
			Content: []interface{}{
				TextContent{Type: "text", Text: "calling tool"},
				ToolUse{Type: "tool_use", ID: "tu_1", Name: "get_weather", Input: map[string]interface{}{"city": "Paris"}},
			},
		},
	}
	got, err := toLibMessages(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Content) != 2 {
		t.Fatalf("expected one message with two blocks, got %+v", got)
	}
	if got[0].Role != llm.RoleAssistant {
		t.Errorf("role = %q, want assistant", got[0].Role)
	}
	if got[0].Content[0].Type != llm.BlockText || got[0].Content[0].Text != "calling tool" {
		t.Errorf("block[0] = %+v", got[0].Content[0])
	}
	block := got[0].Content[1]
	if block.Type != llm.BlockToolUse || block.ToolUse == nil {
		t.Fatalf("block[1] = %+v, want tool_use", block)
	}
	if block.ToolUse.ID != "tu_1" || block.ToolUse.Name != "get_weather" {
		t.Errorf("toolUse = %+v", block.ToolUse)
	}
	var input map[string]interface{}
	if err := json.Unmarshal(block.ToolUse.Input, &input); err != nil {
		t.Fatalf("input unmarshal: %v", err)
	}
	if input["city"] != "Paris" {
		t.Errorf("input = %+v, want city=Paris", input)
	}
}

func TestToLibMessages_InterfaceSliceToolResultPromotesRole(t *testing.T) {
	// A tool-result element inside a []interface{} must still route the
	// enclosing message to RoleTool, exactly like the []ToolResult case.
	in := []Message{
		{
			Role: "user",
			Content: []interface{}{
				ToolResult{Type: "tool_result", ToolUseID: "tu_1", Content: "72F sunny"},
			},
		},
	}
	got, err := toLibMessages(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Content) != 1 {
		t.Fatalf("expected one message with one block, got %+v", got)
	}
	if got[0].Role != llm.RoleTool {
		t.Errorf("expected RoleTool for tool-result element, got %q", got[0].Role)
	}
	block := got[0].Content[0]
	if block.Type != llm.BlockToolResult || block.ToolUseID != "tu_1" || block.Text != "72F sunny" {
		t.Errorf("block = %+v", block)
	}
}

func TestToLibMessages_InterfaceSliceErrorsOnUnknownElement(t *testing.T) {
	in := []Message{
		{Role: "user", Content: []interface{}{42}},
	}
	if _, err := toLibMessages(in); err == nil {
		t.Fatal("expected error for unsupported element type, got nil")
	}
}

func TestToLibMessages_ErrorOnUnknownContent(t *testing.T) {
	in := []Message{
		{Role: "user", Content: 42},
	}
	if _, err := toLibMessages(in); err == nil {
		t.Fatal("expected error for unsupported content type, got nil")
	}
}

func TestToLibTools_FromMCP(t *testing.T) {
	type mcpToolShape struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema struct {
			Type       string                 `json:"type"`
			Properties map[string]interface{} `json:"properties"`
			Required   []string               `json:"required"`
		} `json:"inputSchema"`
	}

	mcpTool := mcpToolShape{
		Name:        "get_weather",
		Description: "Fetch weather",
	}
	mcpTool.InputSchema.Type = "object"
	mcpTool.InputSchema.Properties = map[string]interface{}{
		"city": map[string]interface{}{"type": "string"},
	}
	mcpTool.InputSchema.Required = []string{"city"}

	got, err := toLibTools([]mcpToolShape{mcpTool})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(got))
	}
	if got[0].Name != "get_weather" || got[0].Description != "Fetch weather" {
		t.Errorf("tool fields wrong: %+v", got[0])
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(got[0].InputSchema, &schema); err != nil {
		t.Fatalf("schema unmarshal: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
}

func TestToLibTools_NilOrEmpty(t *testing.T) {
	if got, err := toLibTools(nil); err != nil || got != nil {
		t.Errorf("toLibTools(nil) = (%v, %v), want (nil, nil)", got, err)
	}
}

func TestFromLibContent_Text(t *testing.T) {
	in := []llm.ContentBlock{
		{Type: llm.BlockText, Text: "hi"},
	}
	got := fromLibContent(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	tc, ok := got[0].(TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", got[0])
	}
	if tc.Type != "text" || tc.Text != "hi" {
		t.Errorf("got %+v", tc)
	}
}

func TestFromLibContent_ToolUse(t *testing.T) {
	in := []llm.ContentBlock{
		{
			Type: llm.BlockToolUse,
			ToolUse: &llm.ToolUse{
				ID:    "tu_1",
				Name:  "get_weather",
				Input: json.RawMessage(`{"city":"London"}`),
			},
		},
	}
	got := fromLibContent(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	tu, ok := got[0].(ToolUse)
	if !ok {
		t.Fatalf("expected ToolUse, got %T", got[0])
	}
	if tu.ID != "tu_1" || tu.Name != "get_weather" {
		t.Errorf("tool = %+v", tu)
	}
	if tu.Input["city"] != "London" {
		t.Errorf("input = %+v", tu.Input)
	}
}
