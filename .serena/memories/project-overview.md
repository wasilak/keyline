# Keyline Project Overview

Keyline is a unified authentication proxy service that provides dual authentication modes (OIDC and Basic Auth) simultaneously, supports multiple deployment modes (forwardAuth, auth_request, standalone proxy), and automatically injects Elasticsearch credentials into authenticated requests.

## Purpose

Keyline acts as an authentication gateway between users and Elasticsearch:
- Adds authentication to Elasticsearch without modifying applications
- Automatically creates ES users for all authenticated users with role-based access control
- Supports both interactive users (OIDC) and API clients (Basic Auth)
- Enables horizontal scaling with Redis-backed credential caching
- Centralizes authentication and audit logging for Elasticsearch access

## Tech Stack

- **Language**: Go 1.26
- **Web Framework**: Echo v4
- **Configuration**: Viper
- **Observability**: 
  - Prometheus metrics
  - OpenTelemetry tracing
  - Structured logging (log/slog)
- **Session Storage**: Redis or in-memory
- **Authentication**: OIDC with PKCE, bcrypt password hashing
- **Encryption**: AES-256-GCM for credential caching
- **Build**: Makefile, Docker, go modules

## Code Style and Conventions

### Go Conventions
- **Package names**: Short, lowercase, no underscores (auth, session, config, transport)
- **Type names**: PascalCase for exported (SessionManager, AuthProvider), camelCase for unexported
- **Function names**: camelCase (NewServer, validateConfig)
- **Constants**: PascalCase for exported, camelCase for unexported
- **Avoid stuttering**: Use `auth.Provider` not `auth.AuthProvider`

### Brand-Agnostic Naming
- Function/struct names should NOT include "Keyline" prefix
- Use generic terms: `Provider`, `Server`, `Config` not `KeylineProvider`, `KeylineServer`
- Project name only in documentation, README, binary name, and user-facing messages

### Project Structure
```
keyline/
├── cmd/              # Entry points (cmd/keyline/main.go)
├── internal/         # Private application code
│   ├── auth/         # Authentication providers (OIDC, local)
│   ├── session/      # Session management
│   ├── cache/        # Credential caching (Redis/memory)
│   ├── config/       # Configuration loading/validation
│   ├── transport/    # HTTP transport adapters
│   ├── elasticsearch/ # ES client wrapper
│   ├── usermgmt/     # Dynamic user management
│   ├── server/       # Main server logic
│   ├── observability/ # Metrics, tracing, logging
│   └── state/        # State token management
├── pkg/              # Public libraries (pkg/crypto/)
├── config/           # Example configurations
├── docs/             # Documentation (Docusaurus)
├── integration/      # Integration tests
├── docker-compose*.yml # Docker configurations
├── Makefile          # Build and test commands
└── keyline           # Binary name
```

## Commands Reference

### Build
```bash
task build           # Build binary to bin/keyline
task release:build:target GOOS=linux GOARCH=amd64  # Build for specific platform
task release:build:target GOOS=darwin GOARCH=amd64 # Build for macOS
```

### Test
```bash
task test            # Run unit tests with race detection
go test -v -tags=integration ./integration/...  # Integration tests
go test -v -tags=property ./...                 # Property-based tests
go test -coverprofile=coverage.out ./...        # Generate coverage report
```

### Lint and Format
```bash
task lint            # Run linters (go vet, fmt check)
task format          # Format code (gofmt, goimports)
```

### Development
```bash
task run             # Run with example config
LOG_LEVEL=debug task run  # Run with debug logging
task clean           # Clean build artifacts
task docker:build    # Build Docker image
task docker:build:multiarch  # Build multi-arch Docker image
```

### CI
```bash
task ci              # Run lint + all tests
task ci:backend      # Run backend CI checks only
```

## Key Design Patterns

### Authentication Flow
1. Unauthenticated request → redirect to OIDC or Basic Auth check
2. OIDC with PKCE for secure authorization code flow
3. Session creation with cryptographically random ID
4. Secure cookie (HttpOnly, Secure, SameSite)
5. Role-based credential injection from ES

### Deployment Modes
- **forwardAuth**: Works with Traefik/Nginx reverse proxies
- **standalone**: Keyline proxies all requests directly to ES
- **auth_request**: Nginx auth_request模式

### Security Features
- PKCE prevents authorization code interception
- Secure cookies prevent XSS and CSRF
- Bcrypt password hashing with configurable cost
- AES-256-GCM credential encryption
- No plaintext secrets in logs

## Project Status

✅ Production Ready - All phases complete, comprehensive testing, ready for deployment
