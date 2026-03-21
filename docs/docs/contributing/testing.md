---
sidebar_label: Testing
sidebar_position: 2
---

# Testing

Guide for testing Keyline.

## Unit Tests

```bash
# Run all tests
make test

# Run with race detection
go test -race ./...
```

## Integration Tests

```bash
# Run integration tests (requires Docker)
make test-integration
```

## Property-Based Tests

```bash
# Run property-based tests
make test-property
```

## Coverage

```bash
# Generate coverage report
make coverage
```

## Next Steps

- **[Development](./development.md)** - Development guide
- **[Release Process](./release-process.md)** - Release procedure
- **[Security Reports](./security-reports.md)** - Reporting vulnerabilities
