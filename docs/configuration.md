# Keyline Configuration Reference

Complete reference for all Keyline configuration options.

## Configuration File Format

Keyline uses YAML format for configuration. Environment variables can be substituted using `${VAR_NAME}` syntax.

```yaml
server:
  port: 9000
  mode: forward_auth
```

## Environment Variable Substitution

Environment variables are substituted at startup:

```yaml
session_secret: ${SESSION_SECRET}  # Required: must be set
redis_url: ${REDIS_URL:-redis://localhost:6379}  # Optional: default value
```

If a required environment variable is missing, Keyline will fail to start with a descriptive error.

## Configuration Sections

### Server Configuration

Controls HTTP server behavior and deployment mode.

```yaml
server:
  port: 9000
  mode: forward_auth
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent: 1000
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `port` | int | 9000 | HTTP server port |
| `mode` | string | required | Deployment mode: `forward_auth` or `standalone` |
| `read_timeout` | duration | 30s | Maximum duration for reading request |
| `write_timeout` | duration | 30s | Maximum duration for writing response |
| `max_concurrent` | int | 1000 | Maximum concurrent requests (0 = unlimited) |

**Deployment Modes:**

- `forward_auth`: For use with Traefik ForwardAuth or Nginx auth_request
- `standalone`: Standalone reverse proxy mode

### OIDC Configuration

OpenID Connect authentication settings.

```yaml
oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: your-client-id
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
    - profile
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
  default_es_user: readonly
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `enabled` | bool | no | Enable OIDC authentication |
| `issuer_url` | string | if enabled | OIDC provider issuer URL (must be HTTPS) |
| `client_id` | string | if enabled | OAuth2 client ID |
| `client_secret` | string | if enabled | OAuth2 client secret |
| `redirect_url` | string | if enabled | OAuth2 callback URL (must be HTTPS) |
| `scopes` | []string | no | OAuth2 scopes to request |
| `mappings` | []OIDCMapping | no | Claim-to-ES-user mappings |
| `default_es_user` | string | if enabled | Default ES user if no mappings match |

**OIDC Mapping:**

Mappings are evaluated in order. First match wins.

```yaml
mappings:
  - claim: email              # Claim name from ID token
    pattern: "*@admin.com"    # Pattern with wildcard support
    es_user: admin            # ES user to map to
```

Supported patterns:
- Exact match: `john.doe@example.com`
- Wildcard: `*@admin.example.com`
- Wildcard: `admin-*@example.com`

### Local Users Configuration

Basic Authentication with local users.

```yaml
local_users:
  enabled: true
  users:
    - username: ci-pipeline
      password_bcrypt: $2a$10$...
      es_user: ci_user
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `enabled` | bool | no | Enable local user authentication |
| `users` | []LocalUser | if enabled | List of local users |

**LocalUser:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | yes | Username for Basic Auth |
| `password_bcrypt` | string | yes | Bcrypt hash of password |
| `es_user` | string | yes | ES user to map to |

Generate bcrypt hash:
```bash
htpasswd -bnBC 10 "" password | tr -d ':\n'
```

### Session Configuration

Session management settings.

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  cookie_path: /
  session_secret: ${SESSION_SECRET}
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `ttl` | duration | yes | Session time-to-live |
| `cookie_name` | string | yes | Session cookie name |
| `cookie_domain` | string | no | Cookie domain (use `.example.com` for subdomains) |
| `cookie_path` | string | yes | Cookie path |
| `session_secret` | string | yes | Secret for cookie signing (min 32 bytes) |

Generate session secret:
```bash
openssl rand -base64 32
```

### Cache Configuration

Session storage backend.

```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `backend` | string | yes | Cache backend: `redis` or `memory` |
| `redis_url` | string | if redis | Redis connection URL |
| `redis_password` | string | no | Redis password (if not in URL) |
| `redis_db` | int | no | Redis database number (0-15) |

**Redis URL Format:**
```
redis://[:password@]host[:port][/database]
```

**Backend Comparison:**

| Feature | Redis | Memory |
|---------|-------|--------|
| Persistence | Yes | No |
| Multi-instance | Yes | No |
| Production | Recommended | Not recommended |
| Development | Optional | Recommended |

### Elasticsearch Configuration

Elasticsearch user credentials for mapping.

```yaml
elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `users` | []ESUser | yes | List of ES users |

**ESUser:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | yes | ES username |
| `password` | string | yes | ES password |

These credentials are used to authenticate to Elasticsearch after user authentication.

### Upstream Configuration

Upstream service settings (standalone mode only).

```yaml
upstream:
  url: http://kibana:5601
  timeout: 30s
  max_idle_conns: 100
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `url` | string | if standalone | Upstream service URL |
| `timeout` | duration | no | Request timeout |
| `max_idle_conns` | int | no | Maximum idle connections in pool |

### Observability Configuration

Logging, metrics, and tracing settings.

```yaml
observability:
  log_level: info
  log_format: json
  otel_enabled: true
  otel_endpoint: http://otel-collector:4318
  otel_service_name: keyline
  otel_service_version: v1.0.0
  otel_environment: production
  otel_trace_ratio: 1.0
  metrics_enabled: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `log_level` | string | info | Log level: `debug`, `info`, `warn`, `error` |
| `log_format` | string | json | Log format: `json` or `text` |
| `otel_enabled` | bool | false | Enable OpenTelemetry tracing |
| `otel_endpoint` | string | - | OTLP exporter endpoint |
| `otel_service_name` | string | keyline | Service name for traces |
| `otel_service_version` | string | - | Service version for traces |
| `otel_environment` | string | - | Environment name for traces |
| `otel_trace_ratio` | float | 1.0 | Trace sampling ratio (0.0-1.0) |
| `metrics_enabled` | bool | false | Enable Prometheus metrics endpoint |

**Trace Sampling:**
- `1.0` = 100% sampling (all requests traced)
- `0.1` = 10% sampling (1 in 10 requests traced)
- `0.01` = 1% sampling (1 in 100 requests traced)

## Configuration Validation

Keyline validates configuration at startup. Common validation errors:

### Required Fields

- At least one authentication method must be enabled (OIDC or local users)
- If OIDC enabled: `issuer_url`, `client_id`, `client_secret`, `redirect_url` required
- If local users enabled: at least one user required
- Session configuration required
- Cache configuration required
- Elasticsearch users required

### Field Validation

- `session_secret` must be at least 32 bytes
- `password_bcrypt` must be valid bcrypt hash
- `redirect_url` must be valid HTTPS URL
- `issuer_url` must be valid HTTPS URL
- `redis_url` required if `cache.backend=redis`
- `upstream.url` required if `server.mode=standalone`

### Validate Configuration

```bash
keyline --validate-config --config config.yaml
```

## Complete Example

```yaml
server:
  port: 9000
  mode: forward_auth
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent: 1000

oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: your-client-id.apps.googleusercontent.com
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
    - profile
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
    - claim: email
      pattern: "*@example.com"
      es_user: readonly
  default_es_user: readonly

local_users:
  enabled: true
  users:
    - username: monitoring
      password_bcrypt: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      es_user: monitoring_user

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  cookie_path: /
  session_secret: ${SESSION_SECRET}

cache:
  backend: redis
  redis_url: redis://redis:6379
  redis_db: 0

elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
    - username: monitoring_user
      password: ${ES_MONITORING_PASSWORD}

observability:
  log_level: info
  log_format: json
  otel_enabled: true
  otel_endpoint: http://otel-collector:4318
  otel_service_name: keyline
  otel_service_version: v1.0.0
  otel_environment: production
  otel_trace_ratio: 0.1
  metrics_enabled: true
```

## Environment Variables

Required environment variables for the example above:

```bash
export SESSION_SECRET=$(openssl rand -base64 32)
export OIDC_CLIENT_SECRET=your-oidc-client-secret
export ES_ADMIN_PASSWORD=your-es-admin-password
export ES_READONLY_PASSWORD=your-es-readonly-password
export ES_MONITORING_PASSWORD=your-es-monitoring-password
```

## Configuration Loading

Keyline loads configuration in this order:

1. Load configuration file specified by `--config` flag or `CONFIG_FILE` environment variable
2. Substitute environment variables using `${VAR_NAME}` syntax
3. Validate configuration
4. Fail startup if validation fails

## Best Practices

1. **Store secrets in environment variables**, not in configuration files
2. **Use Redis for production** deployments for session persistence
3. **Enable metrics and tracing** for observability
4. **Set appropriate timeouts** based on your upstream service
5. **Use strong session secrets** (minimum 32 bytes)
6. **Rotate secrets regularly** for security
7. **Validate configuration** before deploying
8. **Use HTTPS** for all OIDC URLs
9. **Configure appropriate log levels** (info for production, debug for troubleshooting)
10. **Set trace sampling ratio** based on traffic volume (lower for high traffic)
