# Keyline - Authentication Proxy for Elasticsearch

Keyline is a unified authentication proxy service that provides dual authentication modes (OIDC and Basic Auth) simultaneously, supports multiple deployment modes (forwardAuth, auth_request, standalone proxy), and automatically injects Elasticsearch credentials into authenticated requests.

## Features

- **Dual Authentication**: Support both interactive (OIDC) and programmatic (Basic Auth) access simultaneously
- **Multiple Deployment Modes**: Works with Traefik (forwardAuth), Nginx (auth_request), or as standalone proxy
- **OIDC Support**: Full OpenID Connect implementation with PKCE, auto-discovery, and token validation
- **Session Management**: Redis or in-memory session storage with configurable TTL
- **Credential Mapping**: Automatic mapping of authenticated users to Elasticsearch credentials based on roles
- **Security First**: Cryptographic randomness, secure cookies, HTTPS enforcement, bcrypt password hashing
- **Production Ready**: Built-in observability, health checks, graceful shutdown, and comprehensive testing
- **Observability**: Prometheus metrics, OpenTelemetry tracing, structured logging with context
- **WebSocket Support**: Full WebSocket proxying support for real-time applications

## Architecture

Keyline acts as an authentication gateway between your users and Elasticsearch:

```
┌─────────┐     ┌──────────┐     ┌──────────┐     ┌──────────────┐
│ Browser │────▶│  Traefik │────▶│ Keyline  │────▶│ Elasticsearch│
└─────────┘     │  /Nginx  │     │  Proxy   │     └──────────────┘
                └──────────┘     └──────────┘
                                      │
                                      ▼
                                 ┌─────────┐
                                 │  OIDC   │
                                 │Provider │
                                 └─────────┘
```

**ForwardAuth Mode**: Keyline validates authentication and returns headers to the reverse proxy
**Standalone Mode**: Keyline proxies all requests directly to Elasticsearch

## Use Cases

- **Secure Elasticsearch Access**: Add authentication to Elasticsearch without modifying your application
- **Multi-Tenant Deployments**: Map different users/roles to different Elasticsearch credentials
- **Hybrid Authentication**: Support both interactive users (OIDC) and API clients (Basic Auth)
- **Compliance**: Centralized authentication and audit logging for Elasticsearch access
- **Zero-Trust Architecture**: Enforce authentication at the gateway level

## Quick Start

### Using Docker (Recommended)

The fastest way to get started is using Docker:

```bash
# Pull the image
docker pull keyline/keyline:latest

# Create a configuration file
cat > config.yaml <<EOF
server:
  port: 9000
  mode: forwardAuth

auth:
  oidc:
    enabled: true
    issuerURL: https://your-oidc-provider.com
    clientID: your-client-id
    clientSecret: \${OIDC_CLIENT_SECRET}
    redirectURL: https://auth.example.com/_oauth/callback
    scopes:
      - openid
      - profile
      - email
    claimMappings:
      username: preferred_username
      roles: groups

  credentialMapping:
    - roles: ["admin"]
      esUsername: elastic_admin
      esPassword: \${ES_ADMIN_PASSWORD}
    - roles: ["viewer"]
      esUsername: elastic_viewer
      esPassword: \${ES_VIEWER_PASSWORD}

session:
  store: memory
  maxAge: 3600
  cookie:
    name: keyline_session
    secure: true
    httpOnly: true
    sameSite: lax
EOF

# Run Keyline
docker run -d \
  --name keyline \
  -p 9000:9000 \
  -v $(pwd)/config.yaml:/etc/keyline/config.yaml \
  -e OIDC_CLIENT_SECRET="your-secret" \
  -e ES_ADMIN_PASSWORD="admin-password" \
  -e ES_VIEWER_PASSWORD="viewer-password" \
  keyline/keyline:latest

# Check health
curl http://localhost:9000/_health
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/keyline.git
cd keyline

# Build the binary
make build

# Run with example configuration
./keyline --config config/config.example.yaml

# Or validate configuration without starting
./keyline --validate-config --config config/config.example.yaml
```

## Configuration

Keyline uses a YAML configuration file with support for environment variable substitution.

### Minimal Configuration

```yaml
server:
  port: 9000
  mode: forwardAuth  # or standalone

auth:
  oidc:
    enabled: true
    issuerURL: https://your-oidc-provider.com
    clientID: your-client-id
    clientSecret: ${OIDC_CLIENT_SECRET}
    redirectURL: https://auth.example.com/_oauth/callback

  credentialMapping:
    - roles: ["*"]  # Match all roles
      esUsername: elastic_user
      esPassword: ${ES_PASSWORD}
```

### ForwardAuth Mode (with Traefik)

```yaml
server:
  mode: forwardAuth
  port: 9000
  forwardHeaders:
    - X-Forwarded-User
    - X-Forwarded-Groups
```

Traefik configuration:

```yaml
http:
  middlewares:
    keyline-auth:
      forwardAuth:
        address: "http://keyline:9000/_auth"
        authResponseHeaders:
          - "X-Forwarded-User"
          - "X-Forwarded-Groups"
```

### Standalone Proxy Mode

```yaml
server:
  mode: standalone
  port: 9000
  upstreamURL: http://elasticsearch:9200
  enableWebSocket: true
```

### Redis Session Store

```yaml
session:
  store: redis
  redis:
    address: redis:6379
    password: ${REDIS_PASSWORD}
    db: 0
    tls: false
    keyPrefix: "keyline:session:"
```

### Basic Auth (for API clients)

```yaml
auth:
  localUsers:
    - username: api_client
      passwordHash: $2a$10$...  # bcrypt hash
      roles:
        - api_access
```

Generate password hash:

```bash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

See [Configuration Reference](docs/configuration.md) for all options.

## Deployment

### Kubernetes with Traefik

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keyline
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: keyline
        image: keyline/keyline:latest
        ports:
        - containerPort: 9000
        env:
        - name: OIDC_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: oidc-client-secret
        volumeMounts:
        - name: config
          mountPath: /etc/keyline
        livenessProbe:
          httpGet:
            path: /_health
            port: 9000
        readinessProbe:
          httpGet:
            path: /_health
            port: 9000
      volumes:
      - name: config
        configMap:
          name: keyline-config
---
apiVersion: v1
kind: Service
metadata:
  name: keyline
spec:
  ports:
  - port: 9000
  selector:
    app: keyline
---
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: keyline-auth
spec:
  forwardAuth:
    address: http://keyline:9000/_auth
    authResponseHeaders:
      - X-Forwarded-User
      - X-Forwarded-Groups
```

See [Deployment Guide](docs/deployment.md) for complete examples.

## Monitoring

### Health Check

```bash
curl http://localhost:9000/_health
```

Response:

```json
{
  "status": "healthy",
  "checks": {
    "session_store": "ok",
    "oidc_provider": "ok"
  }
}
```

### Metrics

Prometheus metrics are exposed at `/_metrics`:

```bash
curl http://localhost:9000/_metrics
```

Key metrics:

- `keyline_auth_attempts_total`: Total authentication attempts
- `keyline_auth_successes_total`: Successful authentications
- `keyline_auth_failures_total`: Failed authentications
- `keyline_session_creates_total`: Sessions created
- `keyline_proxy_requests_total`: Proxied requests
- `keyline_proxy_request_duration_seconds`: Request duration histogram

### Tracing

OpenTelemetry tracing can be enabled:

```yaml
tracing:
  enabled: true
  endpoint: http://jaeger:4318
  serviceName: keyline
```

## Development

### Prerequisites

- Go 1.22 or later
- Redis (optional, for testing Redis session store)
- Docker (optional, for integration tests)

### Build

```bash
# Build binary
make build

# Build for multiple platforms
make build-all
```

### Test

```bash
# Run all tests
make test

# Run with race detection
go test -race ./...

# Run integration tests (requires Docker)
make test-integration

# Run property-based tests
make test-property

# Generate coverage report
make coverage
```

### Lint and Format

```bash
# Format code
make fmt

# Run linters
make lint

# Run go vet
go vet ./...
```

### Local Development

```bash
# Run with example configuration
make run

# Run with custom config
./keyline --config myconfig.yaml

# Validate configuration
./keyline --validate-config --config myconfig.yaml

# Enable debug logging
LOG_LEVEL=debug ./keyline --config config.yaml
```

## Security

### Authentication Flow

1. **Unauthenticated Request**: User accesses protected resource
2. **Redirect to OIDC**: Keyline redirects to OIDC provider with PKCE challenge
3. **User Login**: User authenticates with OIDC provider
4. **Callback**: OIDC provider redirects back with authorization code
5. **Token Exchange**: Keyline exchanges code for tokens using PKCE verifier
6. **Session Creation**: Keyline creates session with cryptographically random ID
7. **Cookie Set**: Secure, HttpOnly, SameSite cookie set in browser
8. **Authenticated Requests**: Subsequent requests include session cookie
9. **Credential Injection**: Keyline injects Elasticsearch credentials based on user roles

### Security Features

- **PKCE**: Proof Key for Code Exchange prevents authorization code interception
- **Secure Cookies**: HttpOnly, Secure, SameSite attributes prevent XSS and CSRF
- **Cryptographic Randomness**: Session IDs and state tokens use crypto/rand
- **Bcrypt**: Password hashing with configurable cost (default 10)
- **TLS Enforcement**: OIDC provider connections require HTTPS
- **Token Validation**: JWT signature and claims validation
- **Session Expiration**: Configurable TTL with automatic cleanup
- **No Plaintext Secrets**: Sensitive values never logged

### Threat Model

Keyline protects against:

- **Session Hijacking**: Secure cookies, HTTPS enforcement
- **CSRF**: SameSite cookies, state parameter validation
- **XSS**: HttpOnly cookies prevent JavaScript access
- **Replay Attacks**: State tokens are single-use
- **Brute Force**: Bcrypt timing-safe comparison
- **Token Leakage**: Tokens never logged or exposed

See [Security Best Practices](docs/deployment.md#security) for deployment recommendations.

## Troubleshooting

### Common Issues

**OIDC Discovery Failed**
```bash
# Test OIDC provider connectivity
curl https://your-oidc-provider.com/.well-known/openid-configuration
```

**Session Not Persisting**
- Check cookie domain matches your application domain
- Verify `session.cookie.secure` is `false` for HTTP (dev only)
- Ensure Redis is accessible if using Redis session store

**Authentication Loop**
- Verify redirect URL is registered in OIDC provider
- Check that callback endpoint `/_oauth/callback` is accessible
- Review logs for state parameter validation errors

See [Troubleshooting Guide](docs/troubleshooting.md) for detailed solutions.

## Documentation

- [Configuration Reference](docs/configuration.md) - Complete configuration options
- [Deployment Guide](docs/deployment.md) - Kubernetes, Docker, Traefik, Nginx
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make test lint fmt`
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Project Status

✅ **Production Ready** - All phases complete, comprehensive testing, ready for deployment
