---
sidebar_label: Troubleshooting
sidebar_position: 1
---

# Troubleshooting

Common issues and solutions organized by error message.

## Configuration Errors

### "environment variable X is required"

**Cause**: Environment variable not set

**Solution**:
```bash
# Session secret
export SESSION_SECRET=$(openssl rand -base64 32)

# Encryption key
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)

# ES admin password
export ES_ADMIN_PASSWORD=your-secure-password
```

### "session_secret must be at least 32 bytes"

**Cause**: Session secret too short

**Solution**:
```bash
export SESSION_SECRET=$(openssl rand -base64 32)
```

### "encryption key must be 32 bytes"

**Cause**: Encryption key wrong length

**Solution**:
```bash
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Verify length
echo -n "$CACHE_ENCRYPTION_KEY" | base64 -d | wc -c  # Should be 32
```

### "Invalid bcrypt hash"

**Cause**: Password not hashed with bcrypt

**Solution**:
```bash
# Generate proper bcrypt hash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

### "Admin credentials invalid: 401 Unauthorized"

**Cause**: ES admin credentials wrong or user lacks privileges

**Solution**:
1. Verify ES admin user exists:
   ```bash
   curl -u elastic:password https://localhost:9200/_security/user/keyline_admin
   ```
2. Ensure user has `manage_security` privilege (use `superuser` role)
3. Check password is correct

### "no role mappings matched and no default roles configured"

**Cause**: User has no matching groups and no defaults set

**Solution**:
```yaml
# Add default roles
default_es_roles:
  - viewer
  - kibana_user
```

## Authentication Errors

### "Invalid or expired state token"

**Cause**: Session storage issue or cookie not transmitted

**Solution**:
1. Verify `session_secret` is configured (min 32 bytes)
2. Check session storage is accessible (Redis/memory)
3. Ensure cookies are being transmitted (check browser dev tools)

### "Redirect URI mismatch"

**Cause**: OIDC redirect URL doesn't match provider config

**Solution**:
1. Verify `redirect_url` in config matches exactly what's in OIDC provider
2. Ensure HTTPS is used (required by most providers)
3. Check for trailing slashes

### "Failed to fetch OIDC discovery document"

**Cause**: Can't reach OIDC provider

**Solution**:
```bash
# Test discovery endpoint
curl https://accounts.google.com/.well-known/openid-configuration

# Check network connectivity from Keyline
```

## Deployment Errors

### Container won't start (Docker)

```bash
# Check logs
docker logs keyline

# Validate configuration
docker run --rm -v $(pwd)/config.yaml:/etc/keyline/config.yaml:ro \
  ghcr.io/wasilak/keyline:latest --validate-config --config /etc/keyline/config.yaml
```

### Pod won't start (Kubernetes)

```bash
# Check status
kubectl get pods -n auth

# Describe pod
kubectl describe pod <pod-name> -n auth

# Check logs
kubectl logs <pod-name> -n auth
```

### Service won't start (systemd)

```bash
# Check status
systemctl status keyline

# Check logs
journalctl -u keyline -n 100

# Validate config
keyline --validate-config --config /etc/keyline/config.yaml
```

### "Can't connect to Redis"

```bash
# Test connectivity
redis-cli ping

# Check Redis service
systemctl status redis

# Verify redis_url format
# Correct: redis://localhost:6379
# Correct: redis://user:pass@host:6379/db
```

### "Can't connect to Elasticsearch"

```bash
# Test ES connectivity
curl -u elastic:password https://localhost:9200/_cluster/health

# Check ES is running
systemctl status elasticsearch
```

## Generating Secrets

### Session Secret
```bash
openssl rand -base64 32
```

### Encryption Key (32 bytes for AES-256-GCM)
```bash
openssl rand -base64 32
```

### Bcrypt Password Hash
```bash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

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
✓ Field types: valid
✓ Value constraints: valid
✓ Admin credentials: validated with ES

Configuration is valid.
```
