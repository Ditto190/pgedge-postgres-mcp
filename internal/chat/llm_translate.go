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
	"fmt"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"
)

// toLibMessages converts our chat.Message slice (with interface{} content)
// into the library's typed []llm.Message form. Content may be a plain
// string (legacy form), or a typed slice of TextContent, ToolUse, or
// ToolResult. Unknown content types return an error.
//
// Tool-result messages are emitted with RoleTool so the library can
// route them to the provider-appropriate position in the conversation.
func toLibMessages(in []Message) ([]llm.Message, error) {
	out := make([]llm.Message, 0, len(in))
	for i, m := range in {
		role := llm.Role(m.Role)
		blocks, isToolResult, err := contentToBlocks(m.Content)
		if err != nil {
			return nil, fmt.Errorf("message %d: %w", i, err)
		}
		if isToolResult {
			role = llm.RoleTool
		}
		out = append(out, llm.Message{Role: role, Content: blocks})
	}
	return out, nil
}

// contentToBlocks normalises a Message.Content interface{} into typed
// content blocks. The bool return indicates whether the content was a
// tool-result slice (so the caller can override the role).
func contentToBlocks(content interface{}) ([]llm.ContentBlock, bool, error) {
	switch c := content.(type) {
	case string:
		return []llm.ContentBlock{{Type: llm.BlockText, Text: c}}, false, nil

	case []TextContent:
		blocks := make([]llm.ContentBlock, 0, len(c))
		for _, t := range c {
			blocks = append(blocks, llm.ContentBlock{Type: llm.BlockText, Text: t.Text})
		}
		return blocks, false, nil

	case []ToolUse:
		blocks := make([]llm.ContentBlock, 0, len(c))
		for _, t := range c {
			raw, err := json.Marshal(t.Input)
			if err != nil {
				return nil, false, fmt.Errorf("marshal tool input: %w", err)
			}
			blocks = append(blocks, llm.ContentBlock{
				Type: llm.BlockToolUse,
				ToolUse: &llm.ToolUse{
					ID:    t.ID,
					Name:  t.Name,
					Input: raw,
				},
			})
		}
		return blocks, false, nil

	case []ToolResult:
		blocks := make([]llm.ContentBlock, 0, len(c))
		for _, r := range c {
			text, err := toolResultText(r.Content)
			if err != nil {
				return nil, false, fmt.Errorf("tool result content: %w", err)
			}
			blocks = append(blocks, llm.ContentBlock{
				Type:      llm.BlockToolResult,
				ToolUseID: r.ToolUseID,
				Text:      text,
				IsError:   r.IsError,
			})
		}
		return blocks, true, nil

	case nil:
		return nil, false, nil

	default:
		return nil, false, fmt.Errorf("unsupported content type %T", content)
	}
}

// toolResultText coerces a tool-result Content (typically a string but
// may be a structured value) to a string for the library's text-based
// tool-result block.
func toolResultText(content interface{}) (string, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

// toLibTools accepts our interface{}-typed tools argument (called with
// []mcp.Tool from the chat client and []llmproxy.Tool from the proxy;
// both are structurally identical) and translates it to []llm.Tool.
//
// The argument is taken as interface{} (and round-tripped through JSON)
// to keep this package free of cross-package coupling — exactly the
// same trick the old Chat method used.
func toLibTools(tools interface{}) ([]llm.Tool, error) {
	if tools == nil {
		return nil, nil
	}
	raw, err := json.Marshal(tools)
	if err != nil {
		return nil, fmt.Errorf("marshal tools: %w", err)
	}
	// Tolerate an empty array.
	if string(raw) == "null" || string(raw) == "[]" {
		return nil, nil
	}

	var shim []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"inputSchema"`
	}
	if err := json.Unmarshal(raw, &shim); err != nil {
		return nil, fmt.Errorf("unmarshal tools: %w", err)
	}

	out := make([]llm.Tool, 0, len(shim))
	for _, t := range shim {
		out = append(out, llm.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return out, nil
}

// fromLibContent converts the library's typed []llm.ContentBlock into
// our LLMResponse.Content []interface{} form (TextContent or ToolUse
// values). Block types we do not surface (image, document) are
// skipped — chat tools today don't produce them.
func fromLibContent(in []llm.ContentBlock) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, b := range in {
		switch b.Type {
		case llm.BlockText:
			out = append(out, TextContent{Type: "text", Text: b.Text})
		case llm.BlockToolUse:
			if b.ToolUse == nil {
				continue
			}
			input := map[string]interface{}{}
			if len(b.ToolUse.Input) > 0 {
				// Best-effort: if input is not a JSON object, leave the map empty.
				_ = json.Unmarshal(b.ToolUse.Input, &input)
			}
			out = append(out, ToolUse{
				Type:  "tool_use",
				ID:    b.ToolUse.ID,
				Name:  b.ToolUse.Name,
				Input: input,
			})
		}
	}
	return out
}
