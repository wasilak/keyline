---
sidebar_label: Security Best Practices
sidebar_position: 5
---

# Security Best Practices

Secure your Keyline deployment with these security guidelines, best practices, and hardening recommendations.

## Overview

Security is critical for authentication proxies. This guide covers network security, secrets management, TLS configuration, and monitoring.

## Network Security

### TLS Configuration

#### Enable HTTPS Everywhere

```yaml
# Keyline configuration
server:
  port: 9000
  mode: standalone

# Nginx reverse proxy (recommended for TLS termination)
server {
    listen 443 ssl http2;
    server_name auth.example.com;
    
    ssl_certificate /etc/nginx/ssl/auth.example.com.crt;
    ssl_certificate_key /etc/nginx/ssl/auth.example.com.key;
    
    # Modern TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;
    
    location / {
        proxy_pass http://keyline:9000;
    }
}
```

#### Certificate Management

```bash
# Let's Encrypt certificate
certbot certonly --webroot -w /var/www/html -d auth.example.com

# Auto-renewal (cron)
0 0 1 * * certbot renew --quiet
```

### Network Isolation

#### Docker Network Segmentation

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No external access

services:
  keyline:
    networks:
      - frontend
      - backend
  
  redis:
    networks:
      - backend  # Internal only
  
  elasticsearch:
    networks:
      - backend  # Internal only
```

#### Kubernetes Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: keyline-ingress
spec:
  podSelector:
    matchLabels:
      app: keyline
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 9000
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # OIDC providers
```

## Secrets Management

### Environment Variables

```bash
# ✅ GOOD: Environment variables
export SESSION_SECRET=$(openssl rand -base64 32)
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)

# ❌ BAD: Hardcoded in config
session_secret: "hardcoded-secret"
```

### Docker Secrets

```yaml
version: '3.8'

services:
  keyline:
    image: ghcr.io/wasilak/keyline:latest
    secrets:
      - session_secret
      - encryption_key
      - es_admin_password

secrets:
  session_secret:
    external: true
  encryption_key:
    external: true
  es_admin_password:
    external: true
```

### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keyline-secrets
type: Opaque
stringData:
  SESSION_SECRET: <generate-secure-random>
  CACHE_ENCRYPTION_KEY: <generate-secure-random>
  ES_ADMIN_PASSWORD: <secure-password>
```

### HashiCorp Vault Integration

```yaml
# Kubernetes with Vault agent injector
apiVersion: v1
kind: Pod
metadata:
  name: keyline
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/agent-inject-secret-secrets.env: 'env'
    vault.hashicorp.com/agent-inject-template-env: |
      {{- with secret "secret/data/keyline" -}}
      SESSION_SECRET={{ .data.session_secret }}
      CACHE_ENCRYPTION_KEY={{ .data.encryption_key }}
      {{- end }}
```

## Authentication Security

### Session Security

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  cookie_path: /
  session_secret: ${SESSION_SECRET}  # Min 32 bytes
```

**Cookie Security Attributes:**
- `HttpOnly: true` - Prevents JavaScript access
- `Secure: true` - Requires HTTPS
- `SameSite: Lax` - CSRF protection
- `Path: /` - Cookie scope

### Password Security

```yaml
local_users:
  users:
    - username: admin
      password_bcrypt: ${ADMIN_PASSWORD_BCRYPT}  # Bcrypt hash
```

**Password Requirements:**
- Minimum 16 characters
- Mix of uppercase, lowercase, numbers, symbols
- Bcrypt cost factor: 10-12
- Rotate every 90 days

### OIDC Security

```yaml
oidc:
  enabled: true
  issuer_url: https://accounts.google.com  # HTTPS required
  client_id: ${OIDC_CLIENT_ID}
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback  # HTTPS required
  scopes:
    - openid
    - email
    - profile
```

**Security Checks:**
- Issuer URL must be HTTPS
- Redirect URL must be HTTPS
- PKCE enabled by default
- State token validation

## Credential Encryption

### Cache Encryption

```yaml
cache:
  backend: redis
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # 32 bytes for AES-256-GCM
```

**Key Management:**
- Generate: `openssl rand -base64 32`
- Store: Environment variable or secrets manager
- Rotate: Quarterly
- Never: Commit to version control

## Monitoring & Auditing

### Security Metrics

```prometheus
# Authentication metrics
keyline_auth_attempts_total{status="success|failure"}
keyline_auth_failures_total{reason="invalid_credentials|expired_session"}

# Session metrics
keyline_session_creates_total
keyline_session_invalidates_total

# Cache metrics
keyline_cred_cache_hits_total
keyline_cred_cache_misses_total
keyline_cred_encrypt_failures_total
```

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: keyline-security
    rules:
      - alert: HighAuthFailureRate
        expr: rate(keyline_auth_failures_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High authentication failure rate"
          
      - alert: EncryptionKeyRotationNeeded
        expr: time() - keyline_encryption_key_created > 7776000  # 90 days
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Encryption key rotation overdue"
```

### Audit Logging

```yaml
observability:
  log_level: info
  log_format: json
  
# Log sensitive events
{
  "level": "audit",
  "event": "user_authenticated",
  "username": "user@example.com",
  "method": "oidc",
  "source_ip": "192.168.1.1",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Hardening Checklist

### Infrastructure

- [ ] TLS enabled for all endpoints
- [ ] Network segmentation implemented
- [ ] Firewall rules configured
- [ ] DDoS protection enabled
- [ ] Rate limiting configured

### Application

- [ ] Session secrets rotated quarterly
- [ ] Encryption keys rotated quarterly
- [ ] Passwords meet complexity requirements
- [ ] OIDC redirect URLs validated
- [ ] Security headers configured

### Monitoring

- [ ] Authentication failures monitored
- [ ] Unusual access patterns detected
- [ ] Audit logs enabled
- [ ] Security alerts configured
- [ ] Log retention policy defined

### Operations

- [ ] Secrets management implemented
- [ ] Backup strategy defined
- [ ] Disaster recovery tested
- [ ] Security patches applied
- [ ] Access controls reviewed

## Compliance

### Supported Standards

| Standard | Status | Notes |
|----------|--------|-------|
| **SOC 2** | ✅ Supports | Audit logging, access controls |
| **PCI DSS** | ✅ Supports | Encryption, network segmentation |
| **GDPR** | ✅ Supports | Data minimization, audit trail |
| **HIPAA** | ⚠️ Partial | Additional controls required |

## Incident Response

### Security Incident Procedure

1. **Detect**: Monitor alerts and logs
2. **Contain**: Isolate affected instances
3. **Investigate**: Review audit logs
4. **Remediate**: Fix vulnerability
5. **Recover**: Restore from backup
6. **Learn**: Document and improve

### Key Rotation After Compromise

```bash
# 1. Generate new secrets
export NEW_SESSION_SECRET=$(openssl rand -base64 32)
export NEW_ENCRYPTION_KEY=$(openssl rand -base64 32)

# 2. Update secrets manager
vault kv put secret/keyline session_secret=$NEW_SESSION_SECRET encryption_key=$NEW_ENCRYPTION_KEY

# 3. Rolling restart
kubectl rollout restart deployment/keyline

# 4. Verify
kubectl get pods -l app=keyline
```

## Next Steps

- **[Docker Deployment](./docker.md)** - Secure Docker configuration
- **[Kubernetes Deployment](./kubernetes.md)** - K8s security
- **[Troubleshooting](../troubleshooting.md)** - Common issues
