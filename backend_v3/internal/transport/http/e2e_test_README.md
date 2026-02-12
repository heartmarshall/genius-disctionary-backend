# End-to-End Tests

This directory contains comprehensive end-to-end tests for the transport layer using testcontainers.

## Overview

The e2e tests use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) to spin up a real PostgreSQL database in a Docker container for each test run. This ensures tests run against a real database environment, making them true integration tests.

## Test Structure

- **e2e_test.go**: Test infrastructure and helpers
  - `setupTestApp()`: Sets up testcontainers, runs migrations, initializes services
  - `executeGraphQL()`: Helper to execute GraphQL queries
  - JSON extraction helpers

- **e2e_queries_test.go**: Tests for GraphQL queries
  - Dictionary queries
  - Inbox queries
  - Study queue queries
  - Dashboard stats queries

- **e2e_mutations_test.go**: Tests for GraphQL mutations
  - Create word
  - Update word
  - Delete word
  - Inbox operations
  - Card review operations

- **e2e_errors_test.go**: Error handling tests
  - Not found errors
  - Invalid input errors
  - Validation errors

- **e2e_dataloader_test.go**: DataLoader functionality tests
  - Batching tests
  - N+1 query prevention verification

## Running Tests

### Prerequisites

- Docker must be running (testcontainers requires Docker)
- Go 1.24+ 

### Run all e2e tests

```bash
cd backend_v3
go test -v ./internal/transport/http/... -tags=e2e
```

### Run specific test

```bash
go test -v ./internal/transport/http/... -run TestCreateWord
```

### Run with coverage

```bash
go test -v -cover ./internal/transport/http/...
```

## Test Isolation

Each test:
1. Starts a fresh PostgreSQL container
2. Runs all migrations
3. Executes the test
4. Cleans up the container

This ensures complete test isolation - no test can affect another.

## Performance

- Container startup: ~5-10 seconds (first time, cached after)
- Migration execution: ~1-2 seconds
- Individual test: varies by complexity

Total test suite time: ~2-5 minutes depending on system.

## Debugging

If tests fail:

1. Check Docker is running: `docker ps`
2. Check container logs (testcontainers will show them)
3. Use `-v` flag for verbose output
4. Check migration files are correct
5. Verify GraphQL schema matches test queries

## Adding New Tests

1. Use `setupTestApp(t)` to get a test app instance
2. Use `app.executeGraphQL()` for queries/mutations
3. Use extraction helpers (`extractString`, `extractInt`, etc.) to verify results
4. Always call `app.teardown(t)` in defer

Example:

```go
func TestMyNewFeature(t *testing.T) {
    app := setupTestApp(t)
    defer app.teardown(t)
    
    resp := app.executeGraphQL(t, query, variables)
    require.Empty(t, resp.Errors)
    // ... assertions
}
```

## Notes

- Tests use a separate test database (test_db)
- Migrations are run automatically
- All tests are independent and can run in parallel
- Testcontainers handles container lifecycle automatically

