/*-----------------------------------------------------------
 *
 * pgEdge Postgres MCP Server - Go Backend Architecture Overview
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-----------------------------------------------------------
 */

# Go Backend Architecture Overview

This document provides a comprehensive overview of the Go backend architecture
for the pgEdge Postgres MCP Server project.

## Project Structure

The Go codebase is organized as a single unified project with the following
structure:

### Command Entry Points (`/cmd`)

```
cmd/
├── pgedge-pg-mcp-svr/   # MCP server executable
├── pgedge-pg-mcp-cli/   # CLI client for natural language queries
└── test-config/         # Configuration testing utility
```

### Internal Packages (`/internal`)

All core Go packages are organized under `/internal/`:

```
internal/
├── api/            # Database API operations
├── auth/           # Authentication and authorization
├── chat/           # CLI chat client functionality
├── compactor/      # Conversation compaction/summarization
├── config/         # Configuration management
├── conversations/  # Conversation storage and retrieval
├── crypto/         # Encryption utilities
├── database/       # Database connection management
├── definitions/    # Tool and resource definitions
├── embedding/      # Embedding providers (Ollama, OpenAI, Voyage)
├── llmproxy/       # LLM proxy for web clients
├── logging/        # Logging infrastructure
├── mcp/            # MCP protocol implementation
├── prompts/        # Prompt templates
├── resources/      # MCP resource handlers
├── search/         # Knowledge base search
├── tools/          # MCP tool implementations
└── tsv/            # TSV file handling
```

### Web Client (`/web`)

The React/MUI web client is located in `/web/`.

## Key Responsibilities

### MCP Server (`pgedge-pg-mcp-svr`)

- Serve MCP protocol over HTTP/HTTPS and stdio
- Authenticate users via bearer tokens
- Execute SQL queries and return results
- Provide MCP tools for database operations
- Serve knowledge base resources
- Proxy LLM requests for web clients

### CLI Client (`pgedge-pg-mcp-cli`)

- Interactive natural language interface
- Direct communication with LLM providers (Anthropic, OpenAI, Ollama)
- Conversation management and history
- Local or remote MCP server connection

### Knowledgebase

The MCP server consumes a pre-built `kb.db` file at runtime; the
builder that produces this file lives in the standalone
[`pgedge-ai-kb`](https://github.com/pgEdge/pgedge-ai-kb) project.

## Core Architectural Patterns

### 1. Connection Management

The project uses `pgx/v5/pgxpool` for PostgreSQL connection pooling:

```go
type ClientManager struct {
    clients map[string]*Client
    mu      sync.RWMutex
}

func (m *ClientManager) GetConnection(ctx context.Context, connID string) (
    *pgxpool.Conn, error) {
    m.mu.RLock()
    client, exists := m.clients[connID]
    m.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("connection not found: %s", connID)
    }

    return client.pool.Acquire(ctx)
}
```

**Key Points:**

- Connection pools are managed per-connection configuration
- Thread-safe access via sync.RWMutex
- Context propagation for cancellation and timeouts

### 2. Configuration Management

Configuration is loaded from YAML files and environment variables:

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Auth     AuthConfig     `yaml:"auth"`
    Database DatabaseConfig `yaml:"database"`
    LLM      LLMConfig      `yaml:"llm"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    return &cfg, nil
}
```

### 3. Error Handling

Standard Go error handling with context wrapping:

```go
result, err := performOperation(ctx)
if err != nil {
    return fmt.Errorf("failed to perform operation: %w", err)
}
```

**Key Patterns:**

- Always use `fmt.Errorf` with `%w` to wrap errors
- Provide context in error messages
- Use `defer` for cleanup (connection release, file close)
- Check context cancellation in long-running operations

### 4. Context Management

Contexts are used throughout for cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

conn, err := pool.Acquire(ctx)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        return fmt.Errorf("timeout acquiring connection")
    }
    return fmt.Errorf("failed to acquire connection: %w", err)
}
defer conn.Release()
```

### 5. MCP Protocol Handler

The MCP handler processes JSON-RPC 2.0 requests:

```go
type Handler struct {
    tools     *tools.Registry
    resources *resources.Registry
    prompts   *prompts.Registry
    config    *config.Config
}

func (h *Handler) HandleRequest(ctx context.Context, req *Request) (
    *Response, error) {
    switch req.Method {
    case "tools/list":
        return h.handleToolsList(ctx, req)
    case "tools/call":
        return h.handleToolsCall(ctx, req)
    case "resources/list":
        return h.handleResourcesList(ctx, req)
    case "resources/read":
        return h.handleResourcesRead(ctx, req)
    default:
        return nil, fmt.Errorf("unknown method: %s", req.Method)
    }
}
```

## Dependency Injection

Components receive dependencies through constructors:

```go
func NewHandler(
    tools *tools.Registry,
    resources *resources.Registry,
    config *config.Config,
) *Handler {
    return &Handler{
        tools:     tools,
        resources: resources,
        config:    config,
    }
}
```

**Benefits:**

- Easy to mock dependencies in tests
- Clear component boundaries
- Explicit dependency graph

## Graceful Shutdown

The server implements graceful shutdown:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go startServer()

<-sigChan

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

server.Shutdown(ctx)
```

## Security Considerations

### Authentication

- Bearer token authentication via HTTP Authorization header
- Token validation on each request
- Support for multiple authentication methods

### Authorization

- Tool-level access control
- Resource access restrictions
- Connection isolation per user

### SQL Injection Prevention

- All user input passed as parameterized queries
- Exception: Tools designed to execute user-provided SQL (documented)

## Logging

The project uses a structured logging package in `/internal/logging/`:

```go
logging.Info("Informational message")
logging.Info("Message with context", "key", value, "another", data)
logging.Error("Error occurred")
logging.Error("Operation failed", "error", err, "context", ctx)
```

## Testing Strategy

See `testing-strategy.md` for detailed testing approach.

## Performance Optimizations

### 1. Connection Pooling

- Reuse connections instead of creating new ones
- Configurable pool sizes
- Health checks to detect failed connections

### 2. Embedding Caching

- Cache embeddings to avoid recomputation
- Batch embedding requests where possible

### 3. Knowledge Base Search

- Vector similarity search with pgvector
- Efficient chunking strategies
