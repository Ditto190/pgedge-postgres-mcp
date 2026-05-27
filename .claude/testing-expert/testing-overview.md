# Testing Overview - pgEdge Postgres MCP Server

This document provides a high-level overview of the testing strategy across
the Postgres MCP Server project.

## Project Structure

The Postgres MCP Server consists of:

- **/cmd/**: Entry points (MCP server, CLI)
- **/internal/**: Core packages (mcp, auth, database, tools, resources)
- **/web/**: React-based web application (not yet implemented)

## Test Organization

### Directory Structure

```
pgedge-nla/
├── cmd/
│   ├── pgedge-pg-mcp-svr/      # MCP server entry point
│   └── pgedge-pg-mcp-cli/      # CLI client entry point
│
├── internal/
│   ├── *_test.go               # Unit tests co-located with source
│   ├── database/
│   │   ├── datastore_test.go
│   │   ├── schema_test.go
│   │   └── ...
│   ├── mcp/
│   │   ├── protocol_test.go
│   │   └── handler_test.go
│   ├── tools/
│   │   └── *_test.go
│   ├── resources/
│   │   └── *_test.go
│   └── auth/
│       └── *_test.go
│
├── web/                        # React web client
│
└── test/                       # Integration tests
    ├── integration_test.go     # Integration tests
    ├── http_integration_test.go # HTTP integration tests
    └── mcp_compliance_test.go  # MCP protocol compliance tests
```

### Test Location Conventions

**GoLang Packages (/internal/, /cmd/):**

- Unit tests are co-located with source code in `*_test.go` files
- Follow Go convention: test files in same package as code
- Integration tests may be in separate subdirectory

**React Project (/web/):**

- Tests in `/web/tests/` subdirectory (per project conventions)
- Component tests co-located with components (standard React practice)
- Integration tests in separate directory

**Integration Tests:**

- Located in top-level `/tests/` directory if separate
- Test interactions between MCP server components
- Located in `/test/` directory

## Test Types

### 1. Unit Tests

**Purpose**: Test individual functions or components in isolation

**Characteristics**:
- Fast execution (milliseconds)
- Use mocks for external dependencies
- No database or network calls
- Test all code paths, edge cases, error conditions

**Location**:

- GoLang: Co-located with source (`*_test.go`)
- React: `/web/tests/unit/` or co-located

**Example**: Testing a connection string builder without database access

### 2. Integration Tests

**Purpose**: Test interaction between multiple components within a sub-project

**Characteristics**:
- May use real dependencies (database, filesystem)
- Test data flow and component collaboration
- Slower than unit tests but faster than E2E
- Run in controlled test environment

**Location**:

- GoLang: Co-located or in `/internal/integration/`
- React: `/web/tests/integration/`

**Example**: Testing database schema migrations with real PostgreSQL

### 3. End-to-End (E2E) Tests

**Purpose**: Test complete workflows across all sub-projects

**Characteristics**:
- Test full user scenarios (client → MCP server → database)
- Use real services and databases
- Slowest tests but highest confidence
- Fewer in number but high value

**Location**: `/tests/integration/`

**Example**: Create user via CLI, authenticate, execute MCP tool

## Test Execution

### Running Tests

**From project root:**
```bash
make test           # Run all tests
make test-server    # Run server tests only
make test-client    # Run CLI client tests only
make lint           # Run linter
```

**Go commands:**
```bash
go test ./internal/...          # Run all internal package tests
go test ./internal/mcp/...      # Run MCP package tests
go test -v ./test/...           # Run integration tests
```

### Environment Variables

**PostgreSQL connection** (standard PG variables):
```bash
export PGHOST=localhost
export PGPORT=5432
export PGDATABASE=postgres
export PGUSER=postgres
export PGPASSWORD=password
```

**LLM API key** (for integration tests):
```bash
export PGEDGE_ANTHROPIC_API_KEY=sk-ant-...
```

## Test Frameworks and Tools

### GoLang

**Core Testing**:
- `testing` package (standard library)
- `testify/assert` and `testify/require` for assertions

**Mocking**:
- Interface-based mocking (manual mocks)
- No external mocking framework currently used

**Database Testing**:
- `pgx/v5/pgxpool` for PostgreSQL connections
- Temporary test databases with unique names
- Automatic cleanup unless `TEST_PGEDGE_MCP_KEEP_DB=1`

**Coverage**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

**Linting**:
- `golangci-lint` with custom configuration
- Enabled linters: errcheck, govet, ineffassign, staticcheck, unused, misspell, gosec
- Configuration in `/.golangci.yml`

### React (Planned)

**Testing Library**:
- Jest (test runner)
- React Testing Library (component testing)
- Testing Library User Event (interaction simulation)

**Mocking**:
- MSW (Mock Service Worker) for API mocking
- Jest mocks for modules

**Coverage**:
```bash
npm test -- --coverage
```

## Coverage Goals

**Overall Project**: >80% coverage

**Critical Components**:
- Database operations: >90%
- User management: >90%
- Authentication/authorization: 100%
- Encryption functions: 100%
- Security-sensitive code: 100%

**Acceptable Lower Coverage**:
- Main entry points (main.go): Lower coverage acceptable
- Logging utilities: Lower coverage acceptable
- Configuration parsing: Lower coverage acceptable

## Continuous Integration

### GitHub Actions Workflows

**Server CI** (`.github/workflows/ci-server.yml`):
- Triggers: Push/PR to Go code
- Tests MCP server code
- Runs: `make test`

**CLI Client CI** (`.github/workflows/ci-cli-client.yml`):
- Triggers: Push/PR to CLI code
- Tests CLI client

**Web Client CI** (`.github/workflows/ci-web-client.yml`):
- Triggers: Push/PR to web code
- Tests React web client

### CI Artifacts

- Coverage reports (HTML) uploaded as artifacts
- Retention: 30 days
- Available for download from workflow runs

## Test Database Management

### Database Lifecycle

1. **Creation**: Test creates unique database `pgedge_mcp_test_<timestamp>`
2. **Schema**: Migrations applied automatically
3. **Test Data**: Tests create and clean up their own data
4. **Cleanup**: Database dropped after tests (unless `TEST_PGEDGE_MCP_KEEP_DB=1`)

### Connection Management

**Integration Tests**:
- Use pgx connection pools
- Configure via environment variables or test setup
- Clean up connections after tests

**Unit Tests**:
- Mock database interfaces where possible
- Use test helpers in `/internal/database/test_helpers.go`

## Security Testing Principles

All tests must verify security requirements:

### Input Validation
- Test with malformed inputs
- Test with oversized inputs
- Test with injection attempts (SQL, XSS, command)
- Test with special characters

### Authentication/Authorization
- Test access controls
- Test session isolation
- Test token validation
- Test permission checks

### Data Isolation
- Verify user data separation
- Verify database connection isolation
- Test cross-user access attempts

### Error Handling
- Verify no sensitive data in errors
- Test error message sanitization
- Verify proper logging without leaking secrets

## Code Quality Standards

### Test Requirements

1. **Every function must have tests** (exceptions: trivial getters/setters)
2. **Critical paths need 100% coverage** (security, data integrity)
3. **Use table-driven tests** for multiple scenarios
4. **Test both success and failure paths**
5. **Include edge cases and boundary conditions**

### Test Characteristics

**Good Tests Are**:
- **Independent**: Don't depend on other tests
- **Fast**: Run quickly (especially unit tests)
- **Repeatable**: Same result every time
- **Self-checking**: Assert conditions, don't require manual verification
- **Timely**: Written with or before the code

**Bad Tests Are**:
- Flaky (intermittent failures)
- Dependent on external state
- Slow without reason
- Testing implementation details instead of behavior
- Missing cleanup code

## Best Practices

### 1. Test Isolation

Each test should be completely independent:

```go
func TestFeature(t *testing.T) {
    // Setup test data
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // Run test
    result := doSomething(db)

    // Verify
    assert.Equal(t, expected, result)
}
```

### 2. Cleanup

Always clean up resources:

```go
func TestDatabase(t *testing.T) {
    db, err := NewTestDatabase()
    require.NoError(t, err)
    defer db.Close()  // Always cleanup

    // Test code...
}
```

### 3. Use Helpers

Extract common test setup into helper functions:

```go
func setupTestEnvironment(t *testing.T) *TestEnvironment {
    // Create database, start services, etc.
}

func TestUserCRUD(t *testing.T) {
    env := setupTestEnvironment(t)
    defer env.TeardownTestEnvironment(t)

    // Test code...
}
```

### 4. Meaningful Assertions

Provide context in error messages:

```go
// Good
assert.Equal(t, expectedCount, actualCount,
    "user count mismatch after creation")

// Better
require.Equal(t, 1, len(users),
    "expected exactly 1 user, got %d", len(users))
```

### 5. Table-Driven Tests

Use for testing multiple scenarios:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "test@example.com", false},
        {"empty input", "", true},
        {"invalid format", "notanemail", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v",
                    err, tt.wantErr)
            }
        })
    }
}
```

## Common Patterns

### Skip Tests Conditionally

```go
func TestDatabase(t *testing.T) {
    if os.Getenv("SKIP_DB_TESTS") != "" {
        t.Skip("Skipping database test")
    }
    // Test code...
}
```

### Test Error Cases

```go
func TestInvalidInput(t *testing.T) {
    err := ProcessData(nil)
    require.Error(t, err, "should error on nil input")
    assert.Contains(t, err.Error(), "nil input")
}
```

### Test Concurrency

```go
func TestConcurrentAccess(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Test concurrent operation
        }()
    }
    wg.Wait()
}
```

## Documentation

- **Integration Tests**: `/tests/README.md`
- **This Guide**: `.claude/testing-expert/`

## Related Documents

- `unit-testing.md` - Unit testing patterns and examples
- `integration-testing.md` - Integration test structure
- `test-utilities.md` - Available test helpers
- `database-testing.md` - Database testing approach
- `coverage-and-quality.md` - Coverage and linting
- `writing-tests.md` - Practical guide for new tests
