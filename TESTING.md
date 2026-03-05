# Keyline Testing Guide

This guide shows you how to test Keyline in various configurations.

## Prerequisites

- Go 1.26+ installed
- Docker (optional, for containerized testing)
- curl and jq (for manual testing)

## Quick Start - Standalone Mode with Basic Auth

This is the easiest way to test Keyline locally.

### 1. Build Keyline

```bash
make build
# or
go build -o bin/keyline ./cmd/keyline
```

### 2. Start Keyline

```bash
./bin/keyline --config config/test-config.yaml
```

You should see output like:
```
INFO Server starting mode=standalone port=9000
INFO Cache initialized backend=memory
INFO Server listening address=:9000
```

### 3. Run Tests

In another terminal:

```bash
# Run the test script
./test-keyline.sh

# Or test manually:
curl -u testuser:password http://localhost:9000/get
```

## Test Scenarios

### Scenario 1: Basic Authentication

Test valid credentials:
```bash
curl -v -u testuser:password http://localhost:9000/get
```

Expected:
- HTTP 200 response
- Request proxied to httpbin.org
- Response includes `X-Es-Authorization` header with ES credentials

Test invalid credentials:
```bash
curl -v -u testuser:wrongpassword http://localhost:9000/get
```

Expected:
- HTTP 401 Unauthorized
- `WWW-Authenticate: Basic realm="Keyline"` header

### Scenario 2: Health Check

```bash
curl http://localhost:9000/healthz
```

Expected response:
```json
{
  "status": "healthy",
  "version": "dev"
}
```

### Scenario 3: Metrics

```bash
curl http://localhost:9000/metrics
```

Expected: Prometheus metrics in text format

### Scenario 4: Different HTTP Methods

```bash
# GET
curl -u testuser:password http://localhost:9000/get

# POST
curl -u testuser:password -X POST http://localhost:9000/post \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# PUT
curl -u testuser:password -X PUT http://localhost:9000/put \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# DELETE
curl -u testuser:password -X DELETE http://localhost:9000/delete
```

### Scenario 5: Request/Response Preservation

Verify headers are preserved:
```bash
curl -u testuser:password \
  -H "X-Custom-Header: test-value" \
  http://localhost:9000/headers
```

Check the response to ensure your custom header was forwarded.

## Docker Testing

### Build and Run

```bash
# Build Docker image
docker build -t keyline:latest .

# Run container
docker run -p 9000:9000 \
  -v $(pwd)/config/test-config.yaml:/etc/keyline/config.yaml \
  keyline:latest --config /etc/keyline/config.yaml

# Test
curl -u testuser:password http://localhost:9000/get
```

### Docker Compose with Traefik

```bash
# Start services
docker-compose -f docker-compose.test.yaml up -d

# Test through Traefik
curl -u testuser:password http://whoami.localhost

# Check Traefik dashboard
open http://localhost:8080

# Stop services
docker-compose -f docker-compose.test.yaml down
```

## Integration Tests

Run the full integration test suite:

```bash
# All integration tests
go test -v -tags=integration ./integration/...

# Specific test suites
go test -v -tags=integration ./integration -run TestBasicAuth
go test -v -tags=integration ./integration -run TestForwardAuth
go test -v -tags=integration ./integration -run TestStandaloneProxy
go test -v -tags=integration ./integration -run TestRedisSession
go test -v -tags=integration ./integration -run TestObservability
```

## Unit Tests

```bash
# All unit tests
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Configuration Validation

Test configuration validation:

```bash
# Valid config
./bin/keyline --validate-config --config config/test-config.yaml

# Invalid config (should fail)
./bin/keyline --validate-config --config config/invalid-config.yaml
```

## Testing with Real OIDC Provider

To test with a real OIDC provider (e.g., Okta, Auth0):

1. Create `config/oidc-config.yaml`:

```yaml
server:
  port: 9000
  mode: standalone

oidc:
  enabled: true
  issuer_url: https://your-tenant.okta.com
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: http://localhost:9000/auth/callback
  scopes:
    - openid
    - email
    - profile
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
  default_es_user: readonly

local_users:
  enabled: false

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: localhost
  cookie_path: /
  session_secret: ${SESSION_SECRET}

cache:
  backend: memory

elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}

upstream:
  url: http://localhost:5601  # Kibana
  timeout: 30s

observability:
  log_level: info
  log_format: text
  otel_enabled: false
```

2. Set environment variables:

```bash
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"
export SESSION_SECRET="your-32-byte-secret"
export ES_ADMIN_PASSWORD="admin-password"
export ES_READONLY_PASSWORD="readonly-password"
```

3. Start Keyline:

```bash
./bin/keyline --config config/oidc-config.yaml
```

4. Open browser to `http://localhost:9000` - you should be redirected to OIDC login

## Testing with Redis

Start Redis:
```bash
docker run -d -p 6379:6379 redis:alpine
```

Update config to use Redis:
```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
```

Test session persistence:
```bash
# Start Keyline
./bin/keyline --config config/test-config.yaml

# Authenticate and get session cookie
curl -v -u testuser:password -c cookies.txt http://localhost:9000/get

# Restart Keyline
# Kill and restart the process

# Use saved cookie (should still work)
curl -v -b cookies.txt http://localhost:9000/get
```

## Performance Testing

Basic load test with Apache Bench:

```bash
# Install ab (Apache Bench)
# macOS: brew install httpd
# Ubuntu: apt-get install apache2-utils

# Test with 1000 requests, 10 concurrent
ab -n 1000 -c 10 -A testuser:password http://localhost:9000/get

# Test with keep-alive
ab -n 1000 -c 10 -k -A testuser:password http://localhost:9000/get
```

## Troubleshooting

### Keyline won't start

Check logs for configuration errors:
```bash
./bin/keyline --config config/test-config.yaml 2>&1 | grep ERROR
```

Validate configuration:
```bash
./bin/keyline --validate-config --config config/test-config.yaml
```

### Authentication fails

Check if credentials are correct:
```bash
# The test password is "password"
# Bcrypt hash: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

Enable debug logging:
```yaml
observability:
  log_level: debug
```

### Upstream connection fails

Test upstream directly:
```bash
curl http://httpbin.org/get
```

Check Keyline logs for upstream errors.

## Next Steps

- See [docs/configuration.md](docs/configuration.md) for full configuration reference
- See [docs/deployment.md](docs/deployment.md) for production deployment
- See [docs/troubleshooting.md](docs/troubleshooting.md) for common issues
