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
    - username: admin
      password_bcrypt: $2a$10$...
      groups:
        - admin
        - superusers
      email: admin@example.com
      full_name: Admin User
    
    - username: developer
      password_bcrypt: $2a$10$...
      groups:
        - developers
      email: dev@example.com
      full_name: Developer User
    
    # Legacy format (deprecated)
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
| `groups` | []string | no | User groups for role mapping (recommended) |
| `email` | string | no | User email address |
| `full_name` | string | no | User full name |
| `es_user` | string | no | **Deprecated**: Direct ES user mapping |

**Note**: The `es_user` field is deprecated. Use `groups` with role mappings instead.

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

Session and credential storage backend.

```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `backend` | string | yes | Cache backend: `redis` or `memory` |
| `redis_url` | string | if redis | Redis connection URL |
| `redis_password` | string | no | Redis password (if not in URL) |
| `redis_db` | int | no | Redis database number (0-15) |
| `credential_ttl` | duration | if user_management.enabled | Credential cache TTL (default: 1h) |
| `encryption_key` | string | if user_management.enabled | 32-byte encryption key for cached credentials |

**Redis URL Format:**
```
redis://[:password@]host[:port][/database]
```

**Encryption Key:**
- Must be exactly 32 bytes (256 bits) for AES-256-GCM
- Generate with: `openssl rand -base64 32`
- Store in environment variable, never in config file
- All Keyline instances must use the same key (for Redis)

**Backend Comparison:**

| Feature | Redis | Memory |
|---------|-------|--------|
| Persistence | Yes | No |
| Multi-instance | Yes | No |
| Horizontal scaling | Yes | No |
| Production | Recommended | Not recommended |
| Development | Optional | Recommended |

### Elasticsearch Configuration

Elasticsearch connection and admin credentials for dynamic user management.

```yaml
elasticsearch:
  # Admin credentials for user management API (required if user_management.enabled)
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false
  
  # Legacy: Static user credentials (deprecated, use dynamic user management)
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `admin_user` | string | if user_management.enabled | Admin username for Security API |
| `admin_password` | string | if user_management.enabled | Admin password for Security API |
| `url` | string | if user_management.enabled | Elasticsearch cluster URL |
| `timeout` | duration | no | Request timeout (default: 30s) |
| `insecure_skip_verify` | bool | no | Skip TLS certificate verification (dev only) |
| `users` | []ESUser | if not using user_management | **Deprecated**: Static ES user credentials |

**Admin User Requirements:**
- Must have `manage_security` privilege in Elasticsearch
- Used exclusively for creating/updating ES users via Security API
- Should be different from user credentials

**ESUser (Deprecated):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | yes | ES username |
| `password` | string | yes | ES password |

**Note**: Static user mapping (`elasticsearch.users`) is deprecated. Use dynamic user management instead.

### User Management Configuration

Dynamic Elasticsearch user management settings.

```yaml
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h
```

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `enabled` | bool | no | Enable dynamic user management (default: false) |
| `password_length` | int | no | Generated password length (default: 32, min: 32) |
| `credential_ttl` | duration | no | Credential cache TTL (default: 1h) |

**When enabled:**
- Keyline automatically creates ES users for all authenticated users
- User groups/claims are mapped to ES roles via `role_mappings`
- Credentials are cached with encryption for performance
- Requires `elasticsearch.admin_user` and `elasticsearch.admin_password`
- Requires `cache.encryption_key` for credential encryption

See [User Management Guide](user-management.md) for detailed documentation.

### Role Mappings Configuration

Map user groups/claims to Elasticsearch roles (applies to ALL authentication methods).

```yaml
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user
  
  - claim: groups
    pattern: "*-admins"
    es_roles:
      - superuser
  
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user
```

**RoleMapping:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `claim` | string | yes | Claim/field name to match (e.g., "groups", "email") |
| `pattern` | string | yes | Pattern to match against claim value |
| `es_roles` | []string | yes | ES roles to assign if pattern matches |

**Pattern Matching:**
- **Exact match**: `admin` matches only "admin"
- **Prefix wildcard**: `admin@*` matches "admin@domain1.com", "admin@domain2.com"
- **Suffix wildcard**: `*@example.com` matches "user@example.com", "admin@example.com"
- **Middle wildcard**: `admin@*.com` matches "admin@us.com", "admin@eu.com"

**Mapping Evaluation:**
1. Evaluate ALL role mappings in order
2. Collect ALL matching ES roles (deduplicated)
3. If at least one mapping matched, use collected roles
4. If NO mappings matched AND `default_es_roles` defined, use default roles
5. If NO mappings matched AND `default_es_roles` NOT defined, deny access

**Default Roles:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `default_es_roles` | []string | no | Roles to assign when no mappings match |

**Note**: Default roles are ONLY used when NO mappings match. If any mapping matches, default roles are ignored.

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
- If user management enabled:
  - `elasticsearch.admin_user` and `elasticsearch.admin_password` required
  - `elasticsearch.url` required
  - `cache.encryption_key` required (must be 32 bytes)
  - At least one role mapping OR `default_es_roles` required
- If user management disabled:
  - `elasticsearch.users` required (legacy static mapping)

### Field Validation

- `session_secret` must be at least 32 bytes
- `password_bcrypt` must be valid bcrypt hash
- `redirect_url` must be valid HTTPS URL
- `issuer_url` must be valid HTTPS URL
- `redis_url` required if `cache.backend=redis`
- `upstream.url` required if `server.mode=standalone`
- `cache.encryption_key` must be exactly 32 bytes when base64-decoded
- `role_mappings[].es_roles` must not be empty
- `user_management.password_length` must be at least 32

### Validate Configuration

```bash
keyline --validate-config --config config.yaml
```

## Complete Example

### With Dynamic User Management (Recommended)

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
    - groups

local_users:
  enabled: true
  users:
    - username: admin
      password_bcrypt: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      groups:
        - admin
      email: admin@example.com
      full_name: Admin User
    
    - username: monitoring
      password_bcrypt: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      groups:
        - monitoring
      email: monitoring@example.com
      full_name: Monitoring User

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
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

elasticsearch:
  admin_user: keyline_admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user
  
  - claim: groups
    pattern: "monitoring"
    es_roles:
      - monitoring_user
  
  - claim: email
    pattern: "*@admin.example.com"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user

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

### Legacy Static Mapping (Deprecated)

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

### With Dynamic User Management

Required environment variables:

```bash
export SESSION_SECRET=$(openssl rand -base64 32)
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export OIDC_CLIENT_SECRET=your-oidc-client-secret
export ES_ADMIN_PASSWORD=your-es-admin-password
```

### Legacy Static Mapping

Required environment variables:

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
2. **Use Redis for production** deployments for session persistence and horizontal scaling
3. **Enable dynamic user management** for better accountability and auditing
4. **Use role mappings** instead of static user mappings for flexibility
5. **Secure encryption key** - generate with `openssl rand -base64 32` and store securely
6. **Enable metrics and tracing** for observability
7. **Set appropriate timeouts** based on your upstream service
8. **Use strong session secrets** (minimum 32 bytes)
9. **Rotate secrets regularly** for security (plan for cache invalidation)
10. **Validate configuration** before deploying
11. **Use HTTPS** for all OIDC URLs and Elasticsearch connections
12. **Configure appropriate log levels** (info for production, debug for troubleshooting)
13. **Set trace sampling ratio** based on traffic volume (lower for high traffic)
14. **Test role mappings** before production deployment
15. **Monitor cache hit rate** (target > 95% for optimal performance)
