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
  enabled: true
```

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
```

| Option | Default | Description |
|--------|---------|-------------|
| `port` | 9000 | HTTP server port |
| `mode` | required | `forward_auth` or `standalone` |
| `read_timeout` | 30s | Request read timeout |
| `write_timeout` | 30s | Response write timeout |

### Session Configuration

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  session_secret: ${SESSION_SECRET}  # Min 32 bytes
```

| Option | Default | Description |
|--------|---------|-------------|
| `ttl` | 24h | Session time-to-live |
| `cookie_name` | keyline_session | Session cookie name |
| `cookie_domain` | required | Cookie domain |
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
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # 32 bytes
```

| Option | Default | Description |
|--------|---------|-------------|
| `backend` | memory | `redis` or `memory` |
| `redis_url` | - | Redis connection URL |
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
| `claim` | Yes | Claim name (`groups` or `email`) |
| `pattern` | Yes | Pattern to match (supports `*` wildcard) |
| `es_roles` | Yes | Elasticsearch roles to assign |
| `default_es_roles` | No | Fallback roles if no mappings match |

**Pattern Examples:**
- Exact: `admin`
- Wildcard prefix: `*-developers`
- Wildcard suffix: `admin@*`

### User Management

```yaml
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | false | Enable dynamic user management |
| `password_length` | 32 | Generated password length |
| `credential_ttl` | 1h | Password cache TTL |

### Elasticsearch Configuration

```yaml
elasticsearch:
  admin_user: ${ES_ADMIN_USER}
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
```

| Option | Required | Description |
|--------|----------|-------------|
| `admin_user` | Yes* | Admin user for Security API |
| `admin_password` | Yes* | Admin password |
| `url` | No | ES cluster URL |
| `timeout` | 30s | Request timeout |

*Required when `user_management.enabled` is true

### Upstream Configuration (Standalone Mode)

```yaml
upstream:
  url: http://kibana:5601
  timeout: 30s
```

| Option | Required | Description |
|--------|----------|-------------|
| `url` | Yes* | Upstream service URL |
| `timeout` | 30s | Upstream request timeout |

*Required for `standalone` mode

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
