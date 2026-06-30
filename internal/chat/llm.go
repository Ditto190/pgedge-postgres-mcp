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
	"fmt"
	"os"
	"strings"

	"github.com/pgEdge/pgedge-go-llm-lib/llm"
	"github.com/pgEdge/pgedge-go-llm-lib/llm/provider/anthropic"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/ollama"
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/provider/openai"
)

// Message represents a chat message
type Message struct {
	Role         string                 `json:"role"`
	Content      interface{}            `json:"content"`
	CacheControl map[string]interface{} `json:"cache_control,omitempty"`
}

// ToolUse represents a tool invocation in a message
type ToolUse struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// TextContent represents text content in a message
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Type      string      `json:"type"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"is_error,omitempty"`
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Content    []interface{} // Can be TextContent or ToolUse
	StopReason string
	TokenUsage *TokenUsage `json:"token_usage,omitempty"` // Optional token usage information (only when debug enabled)
}

// TokenUsage holds token usage information for debug purposes
type TokenUsage struct {
	Provider               string  `json:"provider"`
	PromptTokens           int     `json:"prompt_tokens,omitempty"`
	CompletionTokens       int     `json:"completion_tokens,omitempty"`
	TotalTokens            int     `json:"total_tokens,omitempty"`
	CacheCreationTokens    int     `json:"cache_creation_tokens,omitempty"`
	CacheReadTokens        int     `json:"cache_read_tokens,omitempty"`
	CacheSavingsPercentage float64 `json:"cache_savings_percentage,omitempty"`
}

// LLMClient provides a unified interface for different LLM providers
// readOnlySafetyPrompt is appended to the system prompt when the
// database connection is in read-only mode.  It instructs the LLM
// not to attempt any bypass of the read-only transaction setting.
const readOnlySafetyPrompt = `

CRITICAL SECURITY RULE: The database is in READ-ONLY mode. You must NEVER attempt to:
- Modify the transaction_read_only or default_transaction_read_only settings
- Use SET TRANSACTION READ WRITE or any variant
- Use set_config() to change transaction or session read-only settings
- Use DO blocks or PL/pgSQL to bypass read-only restrictions
- Execute any DDL (CREATE, DROP, ALTER) or DML (INSERT, UPDATE, DELETE) statements
Any attempt to bypass read-only mode is a security violation and will be rejected.`

type LLMClient interface {
	// Chat sends messages and available tools to the LLM and returns the response
	Chat(ctx context.Context, messages []Message, tools interface{}) (LLMResponse, error)

	// ListModels returns a list of available models from the provider
	ListModels(ctx context.Context) ([]string, error)

	// SetReadOnlyMode tells the LLM client whether the database
	// connection is in read-only mode so the system prompt can
	// include appropriate safety instructions.
	SetReadOnlyMode(readOnly bool)
}

// --- libClient: pgedge-go-llm-lib-backed implementation ---------------

// libClient is the implementation of LLMClient backed by
// pgedge-go-llm-lib. Constructors below build provider-appropriate
// llm.Options and return a *libClient wrapped in an LLMClient interface.
type libClient struct {
	inner    llm.Client
	provider string
	debug    bool
	readOnly bool
}

// SetReadOnlyMode sets whether the database is in read-only mode.
func (c *libClient) SetReadOnlyMode(readOnly bool) {
	c.readOnly = readOnly
}

// Chat sends messages and tools through the library and translates the
// response back to our LLMResponse shape.
func (c *libClient) Chat(ctx context.Context, messages []Message, tools interface{}) (LLMResponse, error) {
	libMsgs, err := toLibMessages(messages)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("convert messages: %w", err)
	}
	libTools, err := toLibTools(tools)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("convert tools: %w", err)
	}

	req := llm.ChatRequest{
		Messages:     libMsgs,
		Tools:        libTools,
		SystemPrompt: c.systemPrompt(),
	}
	if c.provider == "anthropic" {
		req = anthropic.WithSystemCaching(req)
		if len(libTools) > 0 {
			req = anthropic.WithToolCaching(req)
		}
	}

	resp, err := c.inner.Chat(ctx, req)
	if err != nil {
		return LLMResponse{}, err
	}

	out := LLMResponse{
		Content:    fromLibContent(resp.Content),
		StopReason: string(resp.StopReason),
	}

	if c.debug {
		out.TokenUsage = buildTokenUsage(c.provider, resp.Usage)
		logDebugTokens(c.provider, resp.Usage)
	}

	return out, nil
}

// ListModels delegates to the library.
func (c *libClient) ListModels(ctx context.Context) ([]string, error) {
	return c.inner.ListModels(ctx)
}

// systemPrompt builds the canonical system prompt, appending the
// read-only safety prompt when SetReadOnlyMode(true) has been called.
func (c *libClient) systemPrompt() string {
	s := chatSystemPrompt
	if c.readOnly {
		s += readOnlySafetyPrompt
	}
	return s
}

// chatSystemPrompt is the base system prompt shared across providers.
const chatSystemPrompt = `You are a helpful PostgreSQL database assistant with expert knowledge on PostgreSQL and products from pgEdge with access to MCP tools.

When executing tools:
- Be concise and direct
- Show results without explaining your methodology unless specifically asked
- Base responses ONLY on actual tool results - never make up or guess data
- Format results clearly for the user
- Only use tools when necessary to answer the question`

// buildTokenUsage assembles a *TokenUsage from a library TokenUsage.
// Cache savings percentage is computed against (input + cache_read)
// to match the old wrapper's display semantics.
func buildTokenUsage(provider string, u llm.TokenUsage) *TokenUsage {
	totalInput := u.PromptTokens + u.CacheReadInputTokens
	savePercent := 0.0
	if totalInput > 0 {
		savePercent = float64(u.CacheReadInputTokens) / float64(totalInput) * 100
	}
	return &TokenUsage{
		Provider:               provider,
		PromptTokens:           u.PromptTokens,
		CompletionTokens:       u.CompletionTokens,
		TotalTokens:            u.PromptTokens + u.CompletionTokens,
		CacheCreationTokens:    u.CacheCreationInputTokens,
		CacheReadTokens:        u.CacheReadInputTokens,
		CacheSavingsPercentage: savePercent,
	}
}

// providerDisplay returns a human-friendly capitalisation of a
// provider name, with explicit forms for the providers we know about.
// Unknown values fall back to a simple first-letter upper-case.
func providerDisplay(provider string) string {
	switch provider {
	case "anthropic":
		return "Anthropic"
	case "openai":
		return "OpenAI"
	case "ollama":
		return "Ollama"
	case "gemini":
		return "Gemini"
	case "voyage":
		return "Voyage"
	}
	if provider == "" {
		return provider
	}
	return strings.ToUpper(provider[:1]) + provider[1:]
}

// logDebugTokens prints the same per-call debug line the old hand-rolled
// clients printed; \r\n leads to clear an in-flight spinner line.
func logDebugTokens(provider string, u llm.TokenUsage) {
	pretty := providerDisplay(provider)
	if u.CacheCreationInputTokens > 0 || u.CacheReadInputTokens > 0 {
		percent := 0.0
		total := u.PromptTokens + u.CacheReadInputTokens
		if total > 0 {
			percent = float64(u.CacheReadInputTokens) / float64(total) * 100
		}
		fmt.Fprintf(os.Stderr,
			"\r\n[LLM] [DEBUG] %s - Prompt Cache: Created %d tokens, Read %d tokens (saved ~%.0f%% on input)\n",
			pretty, u.CacheCreationInputTokens, u.CacheReadInputTokens, percent)
	}
	fmt.Fprintf(os.Stderr,
		"\r[LLM] [DEBUG] %s - Tokens: Input %d, Output %d, Total %d\n",
		pretty, u.PromptTokens, u.CompletionTokens, u.PromptTokens+u.CompletionTokens)
}

// --- Constructors (library-backed) ------------------------------------

// NewAnthropicClient creates an Anthropic-backed LLMClient using the
// pgedge-go-llm-lib. baseURL can be empty to use the library default.
func NewAnthropicClient(apiKey, baseURL, model string, maxTokens int, temperature float64, debug bool) (LLMClient, error) {
	opts := llm.Options{
		APIKey:      apiKey,
		Model:       model,
		BaseURL:     baseURL,
		MaxTokens:   llm.Int(maxTokens),
		Temperature: llm.Float(temperature),
	}
	if debug {
		opts.HTTPClient = newTracingHTTPClient("anthropic", model)
	}
	inner, err := llm.NewClient("anthropic", opts)
	if err != nil {
		return nil, fmt.Errorf("create anthropic client: %w", err)
	}
	return &libClient{inner: inner, provider: "anthropic", debug: debug}, nil
}

// NewOpenAIClient creates an OpenAI-backed LLMClient.
func NewOpenAIClient(apiKey, baseURL, model string, maxTokens int, temperature float64, debug bool) (LLMClient, error) {
	opts := llm.Options{
		APIKey:      apiKey,
		Model:       model,
		BaseURL:     baseURL,
		MaxTokens:   llm.Int(maxTokens),
		Temperature: llm.Float(temperature),
	}
	if debug {
		opts.HTTPClient = newTracingHTTPClient("openai", model)
	}
	inner, err := llm.NewClient("openai", opts)
	if err != nil {
		return nil, fmt.Errorf("create openai client: %w", err)
	}
	return &libClient{inner: inner, provider: "openai", debug: debug}, nil
}

// NewOllamaClient creates an Ollama-backed LLMClient. NOTE: the signature
// changes from (LLMClient) to (LLMClient, error) since the library can
// fail at construction (e.g. invalid BaseURL); the two call sites
// (internal/chat/client.go and internal/llmproxy/proxy.go) are updated
// in Task 4.
func NewOllamaClient(baseURL, model string, debug bool) (LLMClient, error) {
	opts := llm.Options{
		Model:   model,
		BaseURL: baseURL,
	}
	if debug {
		opts.HTTPClient = newTracingHTTPClient("ollama", model)
	}
	inner, err := llm.NewClient("ollama", opts)
	if err != nil {
		return nil, fmt.Errorf("create ollama client: %w", err)
	}
	return &libClient{inner: inner, provider: "ollama", debug: debug}, nil
}
