---
sidebar_label: Integrations
sidebar_position: 1
---

# Integrations

Keyline integrates with Elasticsearch, Kibana, Redis, and OIDC providers.

## Elasticsearch

Keyline works with Elasticsearch 7.x, 8.x, and 9.x, as well as OpenSearch.

### Configuration

```yaml
elasticsearch:
  admin_user: ${ES_ADMIN_USER}
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s
```

### Requirements

- Security API must be enabled
- Admin user must have `manage_security` privilege
- TLS recommended for production

## Kibana

Keyline can proxy requests to Kibana in standalone mode.

### Configuration

```yaml
upstream:
  url: http://kibana:5601
  timeout: 30s
```

### ForwardAuth Mode

When using ForwardAuth, Kibana receives the `Authorization` header from Keyline via the reverse proxy.

## Redis

Redis provides persistent session and credential caching for production deployments.

### Configuration

```yaml
cache:
  backend: redis
  redis_url: redis://localhost:6379
  redis_password: ${REDIS_PASSWORD}
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

### Requirements

- Redis 6.0+ recommended
- TLS recommended for production
- Use managed Redis (ElastiCache, Memorystore) for production

## OIDC Providers

Keyline works with any OIDC-compliant identity provider:

- Google Workspace
- Azure AD (Entra ID)
- Okta
- Auth0
- Keycloak
- Generic OIDC providers

See [OIDC Authentication](./authentication/oidc-authentication.md) for provider-specific setup.
