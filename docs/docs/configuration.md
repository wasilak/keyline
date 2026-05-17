---
sidebar_label: Configuration
sidebar_position: 1
---

# Configuration Guide

Complete guide to configuring Keyline.

## Overview

Keyline uses YAML format for configuration with support for environment variable substitution using `${VAR_NAME}` syntax.

:::warning

If a required environment variable is missing, Keyline will fail to start with a descriptive error.

:::

## Quick Configuration

### Minimal Development Setup

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
  url: http://localhost:9200

user_management:
  password_length: 32
  credential_ttl: 1h
```

> User management activates automatically when `elasticsearch.admin_user` and `elasticsearch.admin_password` are configured.

### Required Environment Variables

```bash
# Generate and set these before starting
export SESSION_SECRET=$(openssl rand -base64 32)
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export ES_ADMIN_PASSWORD=your-es-admin-password
export ADMIN_PASSWORD_BCRYPT=$(htpasswd -bnBC 10 "" admin-password | tr -d ':\n')
```

## Configuration Sections

### Server Configuration

```yaml
server:
  port: 9000
  mode: forward_auth  # or 'standalone'
  read_timeout: 30s
  write_timeout: 30s
  max_concurrent: 0
```

| Option | Default | Description |
|--------|---------|-------------|
| `port` | 9000 | HTTP server port |
| `mode` | required | `forward_auth` or `standalone` |
| `read_timeout` | 30s | Request read timeout |
| `write_timeout` | 30s | Response write timeout |
| `max_concurrent` | 0 (unlimited) | Max concurrent requests; 503 when exceeded |

### CORS Configuration

CORS (Cross-Origin Resource Sharing) controls which browser origins may make requests to Keyline. Configure it as a subsection of `server`.

```yaml
server:
  cors:
    allowed_origins:
      - "https://kibana.yourdomain.com"
```

| Option | Default | Description |
|--------|---------|-------------|
| `cors.allowed_origins` | `[]` (all cross-origin requests rejected) | List of allowed origin domains for CORS requests |

:::warning

Do not use wildcard `*` in production — it weakens CSRF protection.

:::

### Session Configuration

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  cookie_path: /
  session_secret: ${SESSION_SECRET}  # Min 32 bytes
```

| Option | Default | Description |
|--------|---------|-------------|
| `ttl` | 24h | Session time-to-live |
| `cookie_name` | keyline_session | Session cookie name |
| `cookie_domain` | required | Cookie domain |
| `cookie_path` | `/` | Cookie path |
| `session_secret` | required | Secret for cookie signing (min 32 bytes) |

**Generate session secret:**
```bash
openssl rand -base64 32
```

### Cache Configuration

```yaml
cache:
  backend: redis  # or 'memory'
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  redis_db: 0
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # 32 bytes
```

| Option | Default | Description |
|--------|---------|-------------|
| `backend` | memory | `redis` or `memory` |
| `redis_url` | - | Redis connection URL |
| `redis_password` | - | Redis authentication password |
| `redis_db` | 0 | Redis database number (0-15) |
| `credential_ttl` | 1h | Password cache TTL |
| `encryption_key` | required | 32-byte key for AES-256-GCM |

**Generate encryption key:**
```bash
openssl rand -base64 32
```

### Local Users (Basic Auth)

```yaml
local_users:
  enabled: true
  users:
    - username: admin
      password_bcrypt: ${ADMIN_PASSWORD_BCRYPT}
      groups:
        - admin
      email: admin@example.com
```

| Option | Required | Description |
|--------|----------|-------------|
| `username` | Yes | Unique username |
| `password_bcrypt` | Yes | Bcrypt-hashed password |
| `groups` | No | User groups for role mapping |
| `email` | No | User email address |

**Generate bcrypt hash:**
```bash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

### LDAP Configuration

LDAP authentication was added in v2.0 and supports Active Directory, OpenLDAP, and any RFC 4511-compliant directory.

```yaml
ldap:
  enabled: true
  url: ${LDAP_URL}
  bind_dn: ${LDAP_BIND_DN}
  bind_password: ${LDAP_BIND_PASSWORD}
  connection_timeout: 10s
  tls_mode: ldaps
  tls_skip_verify: false
  search_base: ${LDAP_SEARCH_BASE}
  search_filter: "(sAMAccountName={username})"
  group_search_base: ${LDAP_GROUP_SEARCH_BASE}
  group_search_filter: "(member={user_dn})"
  username_attribute: sAMAccountName
  email_attribute: mail
  display_name_attribute: displayName
  group_name_attribute: cn
  required_groups:
    - keyline-users
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | false | Enable LDAP authentication |
| `url` | required | LDAP server URL (`ldap://` or `ldaps://`) |
| `bind_dn` | required | Service account distinguished name |
| `bind_password` | required | Service account password — must be a `${ENV_VAR}` reference |
| `connection_timeout` | 10s | Connection timeout |
| `tls_mode` | `none` | TLS mode: `none`, `ldaps`, or `starttls` |
| `tls_skip_verify` | false | Skip TLS certificate verification (dev only) |
| `search_base` | required | Base DN for user search |
| `search_filter` | required | User search filter; `{username}` is replaced with the login name |
| `group_search_base` | - | Base DN for group search (omit to skip group fetching) |
| `group_search_filter` | - | Group search filter; `{user_dn}` is replaced with the user's DN |
| `username_attribute` | `sAMAccountName` | LDAP attribute mapped to the Keyline username |
| `email_attribute` | `mail` | LDAP attribute mapped to email |
| `display_name_attribute` | `displayName` | LDAP attribute mapped to display name |
| `group_name_attribute` | `cn` | LDAP attribute mapped to group name |
| `attribute_mapping` | - | Map to override attribute names (keys: `username`, `email`, `displayName`, `groupName`) |
| `required_groups` | - | User must belong to at least one listed group; empty = all authenticated users allowed |

:::warning Security

`bind_password` must always be supplied as an environment variable reference (e.g. `${LDAP_BIND_PASSWORD}`). Keyline enforces this requirement at startup and will refuse to start if a plaintext password is detected.

:::

**Valid `tls_mode` values:**

| Value | Description |
|-------|-------------|
| `none` | Plain LDAP (port 389) — avoid in production |
| `ldaps` | LDAP over TLS from connection start (port 636) — recommended |
| `starttls` | Upgrade a plain connection to TLS (port 389) |

### Role Mappings

```yaml
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser

default_es_roles:
  - viewer
  - kibana_user
```

| Option | Required | Description |
|--------|----------|-------------|
| `claim` | Yes | Claim name to evaluate (typically `groups` or `email`, but any claim name from the OIDC token is accepted) |
| `pattern` | Yes | Pattern to match (supports `*` wildcard) |
| `es_roles` | Yes | Elasticsearch roles to assign |
| `default_es_roles` | No | Fallback roles if no mappings match |

**Pattern Examples:**
- Exact: `admin`
- Wildcard prefix: `*-developers`
- Wildcard suffix: `admin@*`

### User Management

Dynamic user management creates and updates Elasticsearch users automatically for every authenticated user. User management activates automatically when `elasticsearch.admin_user` and `elasticsearch.admin_password` are configured.

```yaml
user_management:
  password_length: 32
  credential_ttl: 1h
```

| Option | Default | Description |
|--------|---------|-------------|
| `password_length` | 32 | Generated password length |
| `credential_ttl` | 1h | Password cache TTL |

### Elasticsearch Configuration

```yaml
elasticsearch:
  admin_user: ${ES_ADMIN_USER}
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
  insecure_skip_verify: false
```

| Option | Required | Description |
|--------|----------|-------------|
| `admin_user` | Yes* | Admin user for Security API |
| `admin_password` | Yes* | Admin password |
| `url` | No | ES cluster URL |
| `timeout` | 30s | Request timeout |
| `insecure_skip_verify` | false | Skip TLS certificate verification (dev only) |

*Required to activate dynamic user management

### Upstream Configuration (Standalone Mode)

```yaml
upstream:
  url: http://kibana:5601
  timeout: 30s
  max_idle_conns: 100
  insecure_skip_verify: false
```

| Option | Required | Description |
|--------|----------|-------------|
| `url` | Yes* | Upstream service URL |
| `timeout` | 30s | Upstream request timeout |
| `max_idle_conns` | 100 | Maximum idle connections in pool |
| `insecure_skip_verify` | false | Skip TLS certificate verification (dev only) |

*Required for `standalone` mode

### Observability Configuration

```yaml
observability:
  log_level: info
  log_format: json
  otel_enabled: true
  otel_endpoint: http://otel-collector:4318
  otel_service_name: keyline
  otel_service_version: ${VERSION}
  otel_environment: production
  otel_trace_ratio: 1.0
  otel_tls_skip_verify: false
  metrics_enabled: true
```

| Option | Default | Description |
|--------|---------|-------------|
| `log_level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `log_format` | `json` | Log format: `json` or `text` |
| `otel_enabled` | false | Enable OpenTelemetry tracing |
| `otel_endpoint` | - | OTel collector endpoint URL |
| `otel_service_name` | `keyline` | Service name reported to OTel |
| `otel_service_version` | - | Service version reported to OTel |
| `otel_environment` | - | Deployment environment (e.g. `production`) |
| `otel_trace_ratio` | 1.0 | Trace sampling ratio (0.0 to 1.0) |
| `otel_tls_skip_verify` | false | Skip TLS verification for OTel collector (dev only) |
| `metrics_enabled` | false | Enable Prometheus metrics endpoint |

## Validation

Always validate configuration before starting:

```bash
keyline --validate-config --config config.yaml
```

Expected output:
```
✓ YAML syntax: valid
✓ Environment variables: all set
✓ Required fields: all present
✓ Configuration is valid.
```
