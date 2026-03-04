# Keyline - Authentication Proxy for Elasticsearch

Keyline is a unified authentication proxy service that provides dual authentication modes (OIDC and Basic Auth) simultaneously, supports multiple deployment modes (forwardAuth, auth_request, standalone proxy), and automatically injects Elasticsearch credentials into authenticated requests.

## Features

- **Dual Authentication**: Support both interactive (OIDC) and programmatic (Basic Auth) access simultaneously
- **Multiple Deployment Modes**: Works with Traefik (forwardAuth), Nginx (auth_request), or as standalone proxy
- **OIDC Support**: Full OpenID Connect implementation with PKCE, auto-discovery, and token validation
- **Session Management**: Redis or in-memory session storage with configurable TTL
- **Credential Mapping**: Automatic mapping of authenticated users to Elasticsearch credentials
- **Security First**: Cryptographic randomness, secure cookies, HTTPS enforcement, bcrypt password hashing
- **Production Ready**: Built-in observability, health checks, graceful shutdown, and comprehensive testing

## Quick Start

### Prerequisites

- Go 1.22 or later
- Redis (optional, for distributed session storage)
- OIDC provider (e.g., Okta, Auth0, Keycloak)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/keyline.git
cd keyline

# Install dependencies
make deps

# Build the binary
make build

# Run the server
./bin/keyline --config config/config.yaml
```

### Configuration

Create a `config.yaml` file based on `config/config.example.yaml`:

```yaml
server:
  port: 9000
  mode: forward_auth  # or standalone

oidc:
  enabled: true
  issuer_url: https://your-oidc-provider.com
  client_id: your-client-id
  client_secret: your-client-secret
  redirect_url: https://auth.example.com/auth/callback

# ... see config.example.yaml for full configuration
```

### Environment Variables

Keyline supports environment variable substitution in configuration:

```yaml
oidc:
  client_secret: ${OIDC_CLIENT_SECRET}
```

Set the environment variable:

```bash
export OIDC_CLIENT_SECRET="your-secret-here"
```

## Development

### Build

```bash
make build
```

### Test

```bash
# Run all tests
make test

# Run with race detection
go test -race ./...

# Run property-based tests
make property-test

# Generate coverage report
make coverage
```

### Lint

```bash
make lint
```

### Format Code

```bash
make fmt
```

## Deployment

### Docker

```bash
# Build Docker image
make docker-build

# Run container
make docker-run
```

### Kubernetes

See `docs/deployment.md` for Kubernetes deployment instructions.

## Documentation

- [Configuration Reference](docs/configuration.md)
- [Deployment Guide](docs/deployment.md)
- [Troubleshooting](docs/troubleshooting.md)

## License

MIT License - see LICENSE file for details

## Status

🚧 **Work in Progress** - Phase 1 (Core Infrastructure) in development
