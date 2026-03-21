---
sidebar_label: Testing
sidebar_position: 2
---

# Testing

Guide for testing Keyline.

## Unit Tests

```bash
# Run all tests
task test

# Run with race detection
go test -race ./...
```

## Integration Tests

```bash
# Run integration tests (requires Docker)
go test -v -tags=integration ./integration/...
```

## Property-Based Tests

```bash
# Run property-based tests
go test -v -tags=property ./...
```

## Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Next Steps

- **[Development](./development.md)** - Development guide
- **[Release Process](./release-process.md)** - Release procedure
- **[Security Reports](./security-reports.md)** - Reporting vulnerabilities
