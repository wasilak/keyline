# Keyline Testing Guide

## Overview

This guide walks you through testing Keyline's **dynamic Elasticsearch user management** feature across different authentication scenarios.

## Quick Start (5 Minutes)

The fastest way to test dynamic user management:

```bash
# 1. Start Elasticsearch
docker-compose up -d

# 2. Wait for ES to be ready
docker-compose logs -f setup
# Wait for: "All done!"

# 3. Run Keyline with go run (faster than Docker!)
cd /Users/piotrek/git/keyline
ES_PASSWORD=changeme go run ./cmd/keyline --config config/test-config.yaml

# 4. In another terminal, run automated test script
./test-dynamic-user-mgmt.sh

# 5. Watch Keyline logs (in the first terminal)
# Logs appear in real-time
```

If all tests pass, dynamic user management is working! 🎉

**Note:** You can use `go run` for local testing (faster) or `docker build` for production-like testing.

## Automated Test Script

Keyline includes an automated test script that verifies all core functionality:

```bash
# Run the test script
./test-dynamic-user-mgmt.sh

# Expected output:
# ========================================
# Keyline Dynamic User Management Test
# ========================================
# 
# ℹ Test 1: Checking Elasticsearch connectivity...
# ✓ Elasticsearch is running
# ℹ Test 2: Checking Keyline connectivity...
# ✓ Keyline is running
# ℹ Test 3: Testing Basic Auth as testuser...
# ✓ Basic Auth successful for testuser
# ...
# ✓ All critical tests passed!
```

**What the script tests:**

1. ✓ Elasticsearch connectivity
2. ✓ Keyline connectivity
3. ✓ Basic Auth authentication
4. ✓ ES user creation
5. ✓ Role mapping (groups → ES roles)
6. ✓ Default roles
7. ✓ Credential caching
8. ✓ Unauthorized access rejection
9. ✓ Audit log verification

---

## Choose Your Testing Scenario
| **Forward Auth** | Local users + Traefik | `docker-compose-forwardauth.yml` | 9200 | Traefik integration |
| **OIDC** | Mock OIDC provider | `docker-compose-oidc.yml` | 9201 | OIDC authentication |
| **OIDC + Forward Auth** | OIDC + Traefik | `docker-compose-oidc-forwardauth.yml` | 9202 | Full production-like setup |

---

## Scenario 1: Basic Auth with Local Users

This scenario tests dynamic user management with local users (Basic Auth).

### Start Infrastructure

```bash
# Start Elasticsearch
docker-compose up -d

# Wait for ES to be ready (2-3 minutes)
docker-compose logs -f setup
# Wait for: "All done!"
```

### Build and Start Keyline

```bash
# Build Keyline Docker image
docker build -t keyline:latest .

# Start Keyline with test config
docker run -d \
  --name keyline \
  --network keyline-keyline-network \
  -p 9000:9000 \
  -v $(pwd)/config/test-config.yaml:/app/config.yaml \
  -e ES_PASSWORD=${ELASTIC_PASSWORD} \
  keyline:latest

# Watch logs
docker logs -f keyline
```

### Enable User Management

Edit the running config or restart with `user_management.enabled: true`:

```yaml
user_management:
  enabled: true  # Change from false to true
  password_length: 32
  credential_ttl: 1h
```

### Test Authentication

```bash
# Test 1: Authenticate as testuser
curl -v http://localhost:9000/_security/user \
  -u "testuser:password"

# Expected: ES user created dynamically with roles:
# - developer (from "developers" group)
# - kibana_user (from "developers" group)
# - user (from "users" group)
```

### Verify ES User Creation

```bash
# Check ES user was created
curl -k -u elastic:${ELASTIC_PASSWORD} \
  https://localhost:9200/_security/user/testuser

# Expected response:
{
  "testuser": {
    "username": "testuser",
    "roles": ["developer", "kibana_user", "user"],
    "email": "testuser@example.com",
    "full_name": "Test User",
    "metadata": {
      "source": "basic_auth",
      "groups": ["developers", "users"]
    }
  }
}
```

### Test Role Mapping

```bash
# Test admin user (should get superuser role)
curl -v http://localhost:9000/_security/user \
  -u "admin:password"

# Check ES user
curl -k -u elastic:${ELASTIC_PASSWORD} \
  https://localhost:9200/_security/user/admin

# Expected roles: ["superuser"]
```

### Test Cache Behavior

```bash
# First request (cache miss - creates ES user)
time curl -u "testuser:password" http://localhost:9000/_security/user
# Takes ~100-500ms (password generation + ES API call)

# Second request (cache hit - uses cached credentials)
time curl -u "testuser:password" http://localhost:9000/_security/user
# Takes ~10-50ms (cache lookup only)
```

### Verify Audit Logs

```bash
# Check ES audit logs for actual usernames
curl -k -u elastic:${ELASTIC_PASSWORD} \
  "https://localhost:9200/.security-*/_search?pretty" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "match": {
        "authentication_type": "realm"
      }
    }
  }'

# Look for "testuser" and "admin" in audit logs
# NOT a shared account name
```

---

## Scenario 2: OIDC Authentication

This scenario tests dynamic user management with OIDC authentication.

### Start Infrastructure

```bash
# Start Elasticsearch + Mock OIDC provider
docker-compose -f docker-compose-oidc.yml up -d

# Wait for all services
docker-compose -f docker-compose-oidc.yml logs -f setup
docker-compose -f docker-compose-oidc.yml logs -f mock-oauth2
```

### Configure Mock OIDC Provider

```bash
# Access mock OIDC server UI
open http://localhost:8090

# Or configure via API:
curl -X POST http://localhost:8090/admin/config \
  -H "Content-Type: application/json" \
  -d '{
    "issuers": [
      {
        "id": "default",
        "issuer": "http://localhost:8090/default",
        "clients": [
          {
            "id": "keyline-test",
            "secret": "test-secret",
            "redirectUris": ["http://localhost:9000/auth/callback"]
          }
        ],
        "users": [
          {
            "subject": "user1",
            "email": "alice@example.com",
            "name": "Alice Developer",
            "groups": ["developers", "users"],
            "password": "password"
          },
          {
            "subject": "user2",
            "email": "bob@admin.example.com",
            "name": "Bob Admin",
            "groups": ["admin"],
            "password": "password"
          }
        ]
      }
    ]
  }'
```

### Start Keyline with OIDC Config

```bash
# Update config/test-config-oidc.yaml
# Set user_management.enabled: true

# Start Keyline
docker run -d \
  --name keyline-oidc \
  --network keyline-oidc-net \
  -p 9000:9000 \
  -v $(pwd)/config/test-config-oidc.yaml:/app/config.yaml \
  -e ES_PASSWORD=${ELASTIC_PASSWORD} \
  keyline:latest
```

### Test OIDC Flow

```bash
# 1. Get OIDC authorization URL
curl -v http://localhost:9000/auth/login

# Expected: Redirect to http://localhost:8090/default/login
# Follow redirect in browser or use curl with cookies

# 2. Authenticate with mock OIDC
# (Use browser for interactive login)
# Open: http://localhost:9000/auth/login
# Login as: alice@example.com / password

# 3. After successful login, check session
curl -v http://localhost:9000/_security/user \
  -H "Cookie: keyline_session=<session-cookie>"

# Expected: ES user created for alice@example.com
# Roles: developer, kibana_user (from "developers" group)
```

### Verify OIDC User in ES

```bash
# Check ES user created from OIDC
curl -k -u elastic:${ELASTIC_PASSWORD} \
  "https://localhost:9201/_security/user/alice%40example.com"

# Expected:
{
  "alice@example.com": {
    "username": "alice@example.com",
    "roles": ["developer", "kibana_user"],
    "email": "alice@example.com",
    "full_name": "Alice Developer",
    "metadata": {
      "source": "oidc",
      "groups": ["developers", "users"]
    }
  }
}
```

---

## Scenario 3: Forward Auth with Traefik

This scenario tests Keyline as a forward auth middleware for Traefik.

### Start Infrastructure

```bash
# Start Elasticsearch + Traefik
docker-compose -f docker-compose-forwardauth.yml up -d

# Wait for services
docker-compose -f docker-compose-forwardauth.yml logs -f setup
```

### Start Keyline on Host

```bash
# Build Keyline binary
go build -o keyline ./cmd/keyline

# Run Keyline on host (not in Docker)
# This allows Traefik to reach it via host.docker.internal
./keyline --config config/test-config.yaml
```

### Test Forward Auth Flow

```bash
# Access Elasticsearch via Traefik
curl -v http://localhost:8080/_security/user \
  -u "testuser:password" \
  -H "Host: es.localhost"

# Traefik forwards auth request to Keyline
# Keyline validates and returns ES credentials
# Traefik adds Authorization header and forwards to ES
```

### Verify Traefik Integration

```bash
# Check Traefik logs
docker-compose -f docker-compose-forwardauth.yml logs traefik

# Look for:
# - forwardauth request to Keyline
# - Authorization header added to request
```

---

## Verification Checklist

After each scenario, verify:

### User Management

- [ ] ES users created dynamically (check ES Security API)
- [ ] Groups map to correct ES roles
- [ ] Multiple groups → multiple roles (accumulation)
- [ ] Default roles applied when no groups match
- [ ] User metadata includes source, groups, email, full_name

### Caching

- [ ] First request creates ES user (cache miss)
- [ ] Subsequent requests use cached credentials (cache hit)
- [ ] Cache TTL works (credentials expire and regenerate)
- [ ] Passwords encrypted in cache (not plaintext)

### Security

- [ ] Passwords are cryptographically random (32 chars)
- [ ] Passwords never logged
- [ ] Admin credentials validated on startup
- [ ] TLS works (if enabled)
- [ ] ES audit logs show actual usernames

### Observability

- [ ] Structured logs with context
- [ ] Prometheus metrics exposed
- [ ] OpenTelemetry traces (if enabled)
- [ ] Metrics: user_upserts, cache_hits, role_mapping_matches

---

## Troubleshooting

### ES Connection Failed

```bash
# Check ES is running
docker-compose ps

# Check ES logs
docker-compose logs keyline-es01

# Verify connectivity
curl -k -u elastic:${ELASTIC_PASSWORD} https://localhost:9200
```

### User Management Not Working

```bash
# Check user_management.enabled in config
# Must be: user_management.enabled: true

# Check admin credentials
curl -k -u elastic:${ELASTIC_PASSWORD} https://localhost:9200/_security/user

# Check Keyline logs for errors
docker logs keyline | grep -i "user management"
```

### Role Mapping Not Working

```bash
# Verify role_mappings config syntax
# Check groups in auth result match mapping patterns

# Enable debug logging
# observability.log_level: debug

# Check logs for role mapping matches
docker logs keyline | grep "Role mapping matched"
```

### Cache Issues

```bash
# Check cache backend (memory vs redis)
# For Redis: verify Redis is running

# Check encryption key (must be 32 bytes)
echo -n "your-encryption-key" | wc -c

# Clear cache and retry
# (Restart Keyline or wait for TTL expiry)
```

---

## Performance Testing

### Load Test with Apache Bench

```bash
# Install ab (Apache Bench)
# macOS: brew install httpd
# Linux: apt-get install apache2-utils

# Warm up cache (first request)
curl -u testuser:password http://localhost:9000/_security/user > /dev/null

# Run load test (100 requests, 10 concurrent)
ab -n 100 -c 10 \
  -H "Authorization: Basic $(echo -n 'testuser:password' | base64)" \
  http://localhost:9000/_security/user

# Expected: >95% cache hit rate, <50ms avg response time
```

### Verify Cache Hit Rate

```bash
# Check Prometheus metrics (if enabled)
curl http://localhost:9000/metrics | grep keyline_cred_cache

# Expected:
# keyline_cred_cache_hits_total > 95% of total requests
# keyline_cred_cache_misses_total < 5% of total requests
```

---

## Next Steps

After successful testing:

1. **Review migration guide**: See `docs/migration-guide.md`
2. **Configure for production**: See `docs/configuration.md`
3. **Set up monitoring**: See `docs/troubleshooting-user-management.md`
4. **Deploy to production**: See `docs/deployment.md`

---

## Additional Resources

- [Configuration Guide](configuration.md)
- [User Management Guide](user-management.md)
- [Troubleshooting](troubleshooting-user-management.md)
- [Elastauth Evolution](ELASTAUTH-TO-KEYLINE-EVOLUTION.md)
