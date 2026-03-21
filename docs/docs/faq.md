---
sidebar_label: FAQ
sidebar_position: 2
---

# Frequently Asked Questions

## General

### What is Keyline?

Keyline is an authentication proxy for Elasticsearch that provides OIDC and Basic Auth authentication with dynamic user management.

### How does Keyline differ from elastauth?

Keyline is the successor to elastauth with:
- Single service (no Authelia dependency)
- OIDC support
- Enhanced security (AES-256-GCM encryption)
- Better observability

### What authentication methods are supported?

- OIDC (Google, Azure AD, Okta, etc.)
- Basic Auth (local users)
- Both simultaneously

## Configuration

### How do I generate a session secret?

```bash
openssl rand -base64 32
```

### How do I generate a bcrypt password hash?

```bash
htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

### How do I validate my configuration?

```bash
keyline --validate-config --config config.yaml
```

### Can I run multiple Keyline instances?

Yes, use Redis as the cache backend for shared session storage.

## Deployment

### Does Keyline support Kubernetes?

Yes, see [Kubernetes Deployment](./deployment/kubernetes.md).

### How do I upgrade Keyline?

1. Stop current instance
2. Download new version
3. Start new instance
4. Verify health endpoint

### What's the difference between forward_auth and standalone mode?

- **forward_auth**: Returns auth decisions to reverse proxy (Traefik, Nginx)
- **standalone**: Full reverse proxy, proxies requests directly to upstream

## Troubleshooting

### Why am I getting "encryption key must be 32 bytes"?

The encryption key must be exactly 32 bytes (256 bits) for AES-256-GCM. Generate it with:

```bash
openssl rand -base64 32
```

### Why am I getting "no role mappings matched"?

Add `default_es_roles` to your configuration:

```yaml
default_es_roles:
  - viewer
  - kibana_user
```

### Where can I get help?

- [Troubleshooting Guide](./troubleshooting.md)
- [GitHub Issues](https://github.com/wasilak/keyline/issues)
