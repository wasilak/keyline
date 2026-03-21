---
sidebar_label: Configuration Basics
sidebar_position: 4
---

# Configuration Basics

Keyline uses YAML format for configuration with support for environment variable substitution. This guide covers the essential configuration options you need to get started.

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

:::warning

If a required environment variable is missing, Keyline will fail to start with a descriptive error.

:::

## Essential Configuration Sections

### 1. Server Configuration

Controls HTTP server behavior and deployment mode.

```yaml
server:
  port: 9000
  mode: forward_auth  # or 'standalone'
  read_timeout: 30s
  write_timeout: 30s
```

| Option | Default | Description |
|--------|---------|-------------|
| `port` | 9000 | HTTP server port |
| `mode` | required | `forward_auth` or `standalone` |
| `read_timeout` | 30s | Maximum duration for reading request |
| `write_timeout` | 30s | Maximum duration for writing response |

**Deployment Modes:**

- `forward_auth`: For use with Traefik ForwardAuth or Nginx auth_request
- `standalone`: Standalone reverse proxy mode

### 2. Session Management

Configures user session storage and cookies.

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  session_secret: ${SESSION_SECRET}
```

| Option | Default | Description |
|--------|---------|-------------|
| `ttl` | 24h | Session time-to-live |
| `cookie_name` | keyline_session | Session cookie name |
| `cookie_domain` | required | Cookie domain (use `.example.com` for subdomains) |
| `session_secret` | required | Secret for cookie signing (min 32 bytes) |

:::tip

Generate a secure session secret:

```bash
openssl rand -base64 32
```

:::

### 3. Cache Configuration

Configures credential caching backend.

```yaml
cache:
  backend: memory  # or 'redis'
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

| Option | Default | Description |
|--------|---------|-------------|
| `backend` | memory | `memory` or `redis` |
| `credential_ttl` | 1h | How long generated passwords are cached |
| `encryption_key` | required | 32-byte key for AES-256-GCM encryption |

:::warning

The `encryption_key` must be exactly 32 bytes (256 bits) for AES-256-GCM encryption.

:::

### 4. Local Users (Basic Auth)

Defines users for Basic Authentication.

```yaml
local_users:
  enabled: true
  users:
    - username: admin
      password_bcrypt: ${ADMIN_PASSWORD_BCRYPT}
      groups:
        - admin
      email: admin@example.com
      full_name: Admin User
```

| Option | Description |
|--------|-------------|
| `username` | Unique username |
| `password_bcrypt` | Bcrypt-hashed password |
| `groups` | User groups for role mapping |
| `email` | User email address |
| `full_name` | User display name |

:::tip

Generate bcrypt password hashes:

```bash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

:::

### 5. Role Mappings

Maps user groups to Elasticsearch roles.

```yaml
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

  - claim: groups
    pattern: "*-developers"
    es_roles:
      - developer
      - kibana_user

default_es_roles:
  - viewer
  - kibana_user
```

| Option | Description |
|--------|-------------|
| `claim` | Claim name (`groups` or `email`) |
| `pattern` | Pattern to match (supports `*` wildcard) |
| `es_roles` | Elasticsearch roles to assign |

**Pattern Matching:**

- Exact match: `admin`
- Wildcard prefix: `*-developers` matches `backend-developers`
- Wildcard suffix: `admin@*` matches `admin@example.com`
- Wildcard middle: `*@*.com` matches `user@example.com`

### 6. Elasticsearch Configuration

Configures connection to Elasticsearch cluster.

```yaml
elasticsearch:
  admin_user: ${ES_ADMIN_USER}
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false
```

| Option | Description |
|--------|-------------|
| `admin_user` | Admin user for Security API calls |
| `admin_password` | Admin password |
| `url` | Elasticsearch cluster URL |
| `timeout` | Request timeout |
| `insecure_skip_verify` | Skip TLS verification (development only) |

:::warning

The admin user must have the `manage_security` privilege in Elasticsearch.

:::

### 7. Dynamic User Management

Enables automatic ES user creation.

```yaml
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | false | Enable dynamic user management |
| `password_length` | 32 | Generated password length (min 32) |
| `credential_ttl` | 1h | Password cache TTL |

## Minimal Configuration Examples

### Development (Memory Cache, Basic Auth)

```yaml
server:
  port: 9000
  mode: standalone

local_users:
  enabled: true
  users:
    - username: admin
      password_bcrypt: ${ADMIN_PASSWORD_BCRYPT}
      groups:
        - admin

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: localhost
  session_secret: ${SESSION_SECRET}

cache:
  backend: memory
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

elasticsearch:
  admin_user: elastic
  admin_password: ${ES_PASSWORD}
  url: https://localhost:9200
  insecure_skip_verify: true

user_management:
  enabled: true
```

### Production (Redis Cache, OIDC)

```yaml
server:
  port: 9000
  mode: forward_auth

oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  session_secret: ${SESSION_SECRET}

cache:
  backend: redis
  redis_url: ${REDIS_URL}
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}

role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer

elasticsearch:
  admin_user: ${ES_ADMIN_USER}
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200

user_management:
  enabled: true
```

## Configuration Validation

Validate your configuration before starting:

```bash
keyline --validate-config --config config.yaml
```

This checks:
- YAML syntax
- Required fields
- Environment variable substitution
- bcrypt password validity
- URL formats

## Next Steps

- **[Configuration](../configuration.md)** - Complete configuration options
- **[Integrations](../integrations.md)** - Integration guides
- **[Troubleshooting](../troubleshooting.md)** - Common issues
