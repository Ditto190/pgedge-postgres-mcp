# Key Files

This document identifies critical files in the pgEdge Postgres MCP Server
codebase and their purposes.

## Entry Points

| File | Purpose |
|------|---------|
| `/cmd/pgedge-pg-mcp-svr/main.go` | MCP server entry point |
| `/cmd/pgedge-pg-mcp-cli/main.go` | CLI client entry point |
| `/web/src/main.jsx` | React application entry point |
| `/web/src/App.jsx` | React root component |

## Configuration Files

### Build & Development

| File | Purpose |
|------|---------|
| `/Makefile` | Top-level build commands |
| `/go.mod` | Go module dependencies |
| `/web/package.json` | Client dependencies and scripts |
| `/tests/Makefile` | Integration test commands |

### Linting & Quality

| File | Purpose |
|------|---------|
| `/.golangci.yml` | Go linter configuration |
| `/web/vite.config.js` | Vite build configuration |
| `/web/vitest.config.js` | Test configuration |

## Database Schema

| File | Purpose |
|------|---------|
| `/internal/database/schema.go` | **Primary schema definitions** |

The schema.go file is the **source of truth** for all database migrations.
It contains:

- All table definitions
- Index definitions
- Migration version tracking
- Schema upgrade logic

## Authentication & Authorization

| File | Purpose |
|------|---------|
| `/internal/auth/tokens.go` | Token generation and validation |
| `/internal/auth/sessions.go` | Session lifecycle management |
| `/internal/auth/rbac.go` | Role-based access control |
| `/internal/auth/middleware.go` | Auth HTTP middleware |

### Client Auth

| File | Purpose |
|------|---------|
| `/web/src/contexts/AuthContext.jsx` | Auth state management |

## MCP Protocol

| File | Purpose |
|------|---------|
| `/internal/mcp/server.go` | Main MCP request handler |
| `/internal/mcp/types.go` | Protocol types and constants |
| `/internal/mcp/errors.go` | MCP error definitions |
| `/internal/tools/*.go` | Individual tool implementations |
| `/internal/resources/*.go` | Individual resource implementations |

## Database Operations

| File | Purpose |
|------|---------|
| `/internal/database/pool.go` | Connection pool management |
| `/internal/database/connections.go` | Connection CRUD operations |
| `/internal/database/users.go` | User account operations |
| `/internal/database/tokens.go` | Token storage operations |
| `/internal/database/schema.go` | Schema and migrations |

## Configuration

| File | Purpose |
|------|---------|
| `/internal/config/config.go` | Configuration loading |

## Web Client Components

### Core Components

| File | Purpose |
|------|---------|
| `/web/src/App.jsx` | Application root |
| `/web/src/theme/pgedgeTheme.js` | MUI theme configuration |
| `/web/src/components/Header.jsx` | Application header |
| `/web/src/components/ChatInterface.jsx` | Chat interface |

### Feature Components

| File | Purpose |
|------|---------|
| `/web/src/components/MessageList.jsx` | Message display |
| `/web/src/components/MessageInput.jsx` | Message input |
| `/web/src/components/Login.jsx` | Login component |
| `/web/src/contexts/DatabaseContext.jsx` | Database state |

## Test Utilities

| File | Purpose |
|------|---------|
| `/tests/testutil/database.go` | Database test helpers |
| `/tests/testutil/services.go` | Service management for tests |
| `/tests/testutil/cli.go` | CLI execution helpers |
| `/tests/testutil/config.go` | Test configuration |
| `/tests/testutil/common.go` | Common test utilities |

## Documentation

| File | Purpose |
|------|---------|
| `/CLAUDE.md` | Claude Code instructions |
| `/README.md` | Project overview |
| `/docs/index.md` | Documentation entry point |
| `/docs/quickstart/` | Getting started guides |
| `/docs/configuration/` | Configuration reference |
| `/docs/api/` | API documentation |

## CI/CD

| File | Purpose |
|------|---------|
| `/.github/workflows/*.yml` | GitHub Actions workflows |

## Files to Check When...

### Adding a New MCP Tool

1. `/internal/tools/` - Add tool implementation
2. `/internal/mcp/server.go` - Register tool (if needed)
3. `/.claude/mcp-expert/tools-catalog.md` - Update documentation
4. `/docs/` - Update API documentation

### Adding a New Database Table

1. `/internal/database/schema.go` - Add migration
2. `/.claude/postgres-expert/schema-overview.md` - Update docs

### Adding a New React Component

1. `/web/src/components/` - Add component
2. `/web/tests/` - Add tests
3. `/.claude/react-expert/` - Update if new pattern

### Modifying Authentication

1. `/internal/auth/` - Core auth logic
2. `/internal/database/` - Token/session storage
3. `/.claude/golang-expert/authentication-flow.md` - Update docs
4. `/docs/` - Update API documentation
