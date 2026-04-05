# Phase 6: E2E Tests

This directory contains end-to-end tests for the Open-DB9 application.

## Test Structure

```
e2e/
├── docker-compose.test.yml    # Test environment setup
├── suite_test.go              # Main test suite
├── api/                        # API endpoint tests
│   ├── databases_test.go      # Database CRUD tests
│   ├── snapshots_test.go      # Snapshot tests
│   ├── branches_test.go       # Branch tests
│   ├── files_test.go          # File management tests
│   ├── metrics_test.go        # Metrics & observability tests
│   └── schema_test.go         # Schema introspection tests
├── cli/                       # CLI tests
│   ├── cli_test.go            # CLI framework tests
│   └── commands_test.go       # Command tests
└── helpers/
    ├── setup.go               # Test environment setup
    └── assertions.go          # Custom assertions
```

## Running Tests

```bash
# Run all E2E tests
make test-e2e

# Run specific test suite
go test ./e2e/api -v

# Run with coverage
go test ./e2e/... -coverprofile=coverage.out
```

## Test Environment

The E2E tests use a Docker Compose setup that includes:
- PostgreSQL 15
- The db9 API server
- Test containers for isolation

Each test runs in a clean environment with proper setup/teardown.
