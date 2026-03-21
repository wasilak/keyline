---
sidebar_label: Development
sidebar_position: 1
---

# Development

Guide for developing Keyline.

## Prerequisites

- Go 1.22+
- Node.js 24+ (for documentation)
- Docker (for testing)
- Redis (optional, for testing)

## Building

```bash
# Build binary
task build

# Build for all platforms
task release:build:target GOOS=linux GOARCH=amd64
task release:build:target GOOS=darwin GOARCH=amd64
task release:build:target GOOS=windows GOARCH=amd64
```

## Running Locally

```bash
# Run with configuration
./keyline --config config.yaml

# Enable debug logging
LOG_LEVEL=debug ./keyline --config config.yaml
```

## Code Style

- Follow Go best practices
- Run `gofmt` before committing
- Add tests for new features

## Next Steps

- **[Testing](./testing.md)** - Testing guide
- **[Release Process](./release-process.md)** - Release procedure
- **[Security Reports](./security-reports.md)** - Reporting vulnerabilities
