---
sidebar_label: Quick Start
sidebar_position: 3
---

# Quick Start

Get Keyline up and running in 5 minutes. This guide covers the fastest way to get started with Keyline for evaluating its features.

## Prerequisites

- Docker and Docker Compose (recommended), OR
- Keyline binary (for bare-metal installation)
- Access to an Elasticsearch cluster (7.x, 8.x, or 9.x)
- (Optional) OIDC provider credentials (Google, Azure AD, Okta, etc.)

## Option 1: Docker Compose (Recommended)

The fastest way to evaluate Keyline is using Docker Compose with a pre-configured Elasticsearch cluster.

### Step 1: Clone the Repository

```bash
git clone https://github.com/wasilak/keyline.git
cd keyline
```

### Step 2: Set Environment Variables

```bash
# Generate encryption keys
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export SESSION_SECRET=$(openssl rand -base64 32)
export ES_PASSWORD=$(openssl rand -base64 32)
export ELASTIC_PASSWORD=changeme
```

### Step 3: Start Elasticsearch

```bash
docker-compose up -d setup keyline-es01
```

Wait for Elasticsearch to be healthy (about 30-60 seconds):

```bash
docker-compose ps
```

### Step 4: Configure Keyline

Create a `config.yaml` file:

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
      email: admin@example.com
      full_name: Admin User

session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: localhost
  cookie_path: /
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

default_es_roles:
  - viewer

elasticsearch:
  admin_user: elastic
  admin_password: ${ES_PASSWORD}
  url: https://keyline-es01:9200
  timeout: 30s
  insecure_skip_verify: true

user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

upstream:
  url: https://keyline-es01:9200
  timeout: 30s
  insecure_skip_verify: true
```

### Step 5: Start Keyline

```bash
docker-compose up -d keyline
```

### Step 6: Verify Keyline is Running

```bash
curl http://localhost:9000/_health
```

Expected response:

```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

### Step 7: Test Authentication

```bash
curl -u admin:password http://localhost:9000/_cluster/health?pretty
```

## Option 2: Binary Installation

### Step 1: Download Keyline

```bash
# Linux
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-linux-amd64.tar.gz
tar -xzf keyline-linux-amd64.tar.gz
sudo mv keyline /usr/local/bin/

# macOS
curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-darwin-amd64.tar.gz
tar -xzf keyline-darwin-amd64.tar.gz
sudo mv keyline /usr/local/bin/
```

### Step 2: Create Configuration

Create `config.yaml` as shown in Option 1, Step 4.

### Step 3: Run Keyline

```bash
export ADMIN_PASSWORD_BCRYPT=$(htpasswd -bnBC 10 "" password | tr -d ':\n')
export SESSION_SECRET=$(openssl rand -base64 32)
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export ES_PASSWORD=your-es-password

keyline --config config.yaml
```

## Option 3: Quick Evaluation with Test Script

Keyline includes a test script for rapid evaluation:

```bash
# Run the test script
./test-keyline.sh
```

This script:
1. Starts Elasticsearch with Docker
2. Configures Keyline with test users
3. Runs authentication tests
4. Verifies dynamic user management

## What's Next?

Now that Keyline is running:

1. **[Configuration](../configuration.md)** - Learn about configuration options
2. **[Authentication](../authentication/overview.md)** - Configure OIDC or Basic Auth
3. **[User Management](../user-management/dynamic-user-management.md)** - Set up dynamic ES users
4. **[Deployment Modes](../deployment-modes/forwardauth-traefik.md)** - Integrate with your proxy

## Troubleshooting

### Keyline Won't Start

Check logs for errors:

```bash
docker-compose logs keyline
# OR
journalctl -u keyline  # For systemd installations
```

### Can't Connect to Elasticsearch

Verify Elasticsearch is healthy:

```bash
curl -k -u elastic:changeme https://localhost:9200/_cluster/health?pretty
```

### Authentication Fails

1. Verify `session_secret` is at least 32 bytes
2. Check `encryption_key` is exactly 32 bytes (base64)
3. Ensure bcrypt passwords are valid

## Common Configuration Patterns

### Development (Memory Cache)

```yaml
cache:
  backend: memory
```

### Production (Redis Cache)

```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

### ForwardAuth Mode (Traefik)

```yaml
server:
  mode: forward_auth
  port: 9000
```

### Standalone Proxy Mode

```yaml
server:
  mode: standalone
  port: 9000

upstream:
  url: http://kibana:5601
```

## Additional Resources

- **[Configuration](../configuration.md)** - Complete configuration options
- **[Deployment Guide](../deployment/docker.md)** - Production deployment guides
- **[Troubleshooting](../troubleshooting.md)** - Common issues and solutions
