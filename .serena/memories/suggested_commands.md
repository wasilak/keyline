# Keyline Suggested Commands

## Quick Start

```bash
# Build and run
task build && ./bin/keyline --config config/config.example.yaml

# Build Docker image
task docker:build && task docker:build:multiarch
```

## Development Workflow

```bash
# Code changes
1. Edit code in internal/ or pkg/
2. Run lint and format
3. Run tests
4. Build to verify
5. Commit changes

# Verify after changes
task lint
task format
task test
go build -o bin/keyline ./cmd/keyline
./bin/keyline --validate-config --config config/config.example.yaml
```

## Testing

```bash
# Unit tests (always run after code changes)
task test

# With race detection
go test -race ./...

# Property-based tests
go test -v -tags=property ./...

# Integration tests (requires Docker)
go test -v -tags=integration ./integration/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Linting and Formatting

```bash
# Check formatting (fails if not formatted)
gofmt -l -s .

# Format code
task format

# Run linters
task lint

# Manual vet (run if linting shows issues)
go vet ./...
```

## Build and Run

```bash
# Development
task run              # Run with example config
LOG_LEVEL=debug task run  # Run with debug logging (task run doesn't support debug flag)

# Production build
task build            # Binary: bin/keyline
task release:build:target GOOS=linux GOARCH=amd64   # Build for specific platform
task release:build:target GOOS=darwin GOARCH=amd64  # Build for macOS

# Docker
task docker:build     # Build image
task docker:build:multiarch  # Build multi-arch image
```

## Continuous Integration

```bash
# Full CI pipeline
task ci               # lint + all tests
task ci:backend       # backend checks only
```

## Configuration

```bash
# Validate config without starting
./bin/keyline --validate-config --config myconfig.yaml

# Check version
./bin/keyline --version

# Help
./bin/keyline --help
```

## Common Tasks

```bash
# Clean everything
task clean

# Dependencies
go mod download        # Download dependencies
go mod tidy           # Tidy dependencies
go get -u ./...       # Upgrade all dependencies
```

## Git Workflow

```bash
# After completing a task
git add -A
git commit -m "feat: implement X\n\nTask: [description]"
git push
```

## Environment Setup

```bash
# Generate encryption keys
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export SESSION_SECRET=$(openssl rand -base64 32)

# Generate password hash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

## Debugging

```bash
# Debug logging
LOG_LEVEL=debug task run

# Check health
curl http://localhost:9000/_health

# Check metrics
curl http://localhost:9000/_metrics
```
