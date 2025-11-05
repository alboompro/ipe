# Testing Guide

This document describes the test suite for the IPE application.

## Test Structure

The test suite is organized into several categories:

### Unit Tests
- **Location**: Co-located with source files (`*_test.go`)
- **Packages**: `api`, `app`, `channel`, `connection`, `events`, `storage`, `utils`, `websockets`, `concurrency`
- **Requirements**: No external dependencies

### Integration Tests
- **Location**: `integration/integration_test.go`
- **Requirements**: Redis server (optional - tests skip if unavailable)
- **Purpose**: End-to-end scenarios combining WebSocket and HTTP API

### Redis Tests
- **Location**: `redis/redis_test.go`
- **Requirements**: Redis server running
- **Purpose**: Test Redis client functionality for cross-instance scaling

### Webhook Tests
- **Location**: `app/webhooks_test.go`
- **Requirements**: None (uses test HTTP server)
- **Purpose**: Test webhook delivery and payload validation

## Running Tests

### All Tests
```bash
make test
# or
go test ./...
```

### Unit Tests Only
```bash
make test-unit
# or
go test ./api/... ./app/... ./channel/... ...
```

### Integration Tests
Requires Redis to be running:
```bash
# Start Redis (if not running)
docker run -d -p 6379:6379 redis:7-alpine

# Run integration tests
make test-integration
# or
REDIS_HOST=localhost REDIS_PORT=6379 go test ./redis/... ./integration/...
```

### With Race Detector
```bash
make test-race
# or
go test -race ./...
```

### With Coverage
```bash
make test-coverage
# Opens coverage.html in browser
```

### Specific Package
```bash
go test -v ./websockets/...
go test -v ./api/...
```

## Test Coverage

The test suite aims for:
- **Critical paths**: 100% coverage (WebSocket, API handlers, auth)
- **Overall**: >85% code coverage
- **All tests passing**: Required before deployment

View coverage report:
```bash
make test-coverage
open coverage.html
```

## CI/CD

Tests run automatically on:
- Push to main/master/develop branches
- Pull requests
- Scheduled runs (weekly)

### GitHub Actions Workflows

1. **CI Workflow** (`.github/workflows/ci.yml`)
   - Runs all tests with race detector
   - Runs linters
   - Builds for multiple platforms
   - Security scanning

2. **Test Workflow** (`.github/workflows/test.yml`)
   - Separate unit, integration, and webhook test jobs
   - Coverage reporting
   - Test matrix across Go versions

3. **Lint Workflow** (`.github/workflows/lint.yml`)
   - Code formatting checks
   - Static analysis
   - golangci-lint

4. **Build Workflow** (`.github/workflows/build.yml`)
   - Multi-platform builds
   - Automatic releases on tags

5. **CodeQL Workflow** (`.github/workflows/codeql.yml`)
   - Security analysis
   - Vulnerability detection

## Local Development

### Running CI Checks Locally
```bash
make ci
# Runs: fmt-check, vet, lint, test-race
```

### Setup
```bash
# Install dependencies
make deps

# Install test tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# For integration tests, start Redis
docker run -d -p 6379:6379 --name redis-test redis:7-alpine
```

### Running Specific Tests
```bash
# Run a specific test
go test -v -run TestWebSocket_ConnectionEstablishment_ValidProtocol ./websockets/...

# Run tests matching a pattern
go test -v -run "Subscribe" ./app/...

# Run with verbose output
go test -v ./...
```

## Test Utilities

The `testutils` package provides:
- `NewTestApp()` - Create test applications
- `NewTestStorage()` - Create in-memory storage
- `NewWebSocketTestServer()` - Create test WebSocket server
- `ConnectWebSocket()` - Helper for WebSocket connections
- `GetAppFromStorage()` - Retrieve app from storage

The `mocks` package provides:
- `NewMockSocket()` - Mock WebSocket for testing
- Message tracking and verification

## Writing New Tests

1. **Follow naming convention**: `TestFunctionName_Scenario`
2. **Use table-driven tests** for multiple scenarios
3. **Test error cases** as well as success cases
4. **Use subtests** for related test cases
5. **Clean up resources** (connections, channels, etc.)

Example:
```go
func TestFunctionName_Success(t *testing.T) {
    // Arrange
    app := testutils.NewTestApp()
    
    // Act
    result, err := app.DoSomething()
    
    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    if result == nil {
        t.Error("Expected result to be non-nil")
    }
}
```

## Troubleshooting

### Tests failing with Redis errors
- Ensure Redis is running: `redis-cli ping`
- Check environment variables: `REDIS_HOST`, `REDIS_PORT`, `REDIS_DB`
- Integration tests will skip if Redis is unavailable

### Race detector findings
- Run with `-race` flag: `go test -race ./...`
- Fix data races before merging

### Coverage too low
- Run coverage: `make test-coverage`
- Review coverage.html to identify untested code
- Add tests for critical paths

## Best Practices

1. **Isolation**: Each test should be independent
2. **Deterministic**: Tests should produce consistent results
3. **Fast**: Unit tests should run quickly
4. **Clear**: Test names should describe what they test
5. **Comprehensive**: Cover both success and failure paths

