# External Integrations
_Generated: 2026-05-16 | Focus: tech_

## Summary

Keyline integrates with four categories of external systems: authentication backends (LDAP/AD, any OIDC provider), a downstream Elasticsearch cluster (Security API), a cache backend (Redis or in-memory), and an optional OpenTelemetry Collector for traces. All credentials are passed via environment variables; no secrets are stored in config files.

---

## Authentication Backends

### LDAP / Active Directory

**Purpose:** Validate user credentials, fetch groups, and optionally enforce required-group membership.

**Library:** `github.com/go-ldap/ldap/v3` v3.4.13

**Implementation:** `internal/auth/ldap.go` — `LDAPProvider`

**Protocol flow:**
1. Service-account bind (search bind)
2. Search user by configurable filter (default: `sAMAccountName={username}`)
3. User bind (password verification)
4. Re-bind as service account → group search

**TLS modes:** `none` | `ldaps` | `starttls` — configured via `ldap.tls_mode`

**Config keys (all in `ldap:` section of YAML):**
| Key | Env var pattern | Notes |
|-----|----------------|-------|
| `url` | `${LDAP_URL}` | Must start with `ldap://` or `ldaps://` |
| `bind_dn` | `${LDAP_BIND_DN}` | Service account DN |
| `bind_password` | `${LDAP_BIND_PASSWORD}` | **Must** be an `${ENV_VAR}` reference; literal values rejected |
| `search_base` | `${LDAP_SEARCH_BASE}` | e.g. `DC=corp,DC=example,DC=com` |
| `search_filter` | — | e.g. `(sAMAccountName={username})` |
| `group_search_base` | `${LDAP_GROUP_SEARCH_BASE}` | Optional |
| `group_search_filter` | — | e.g. `(member={user_dn})` |
| `tls_skip_verify` | — | `false` in production |
| `connection_timeout` | — | Default: `10s` |

**Attribute mapping:** Four named attributes (`username_attribute`, `email_attribute`, `display_name_attribute`, `group_name_attribute`) plus an `attribute_mapping` map that overrides them. Defaults match Active Directory (`sAMAccountName`, `mail`, `displayName`, `cn`).

**Username normalization:** After attribute lookup, the username is lowercased, trimmed, and non-`[a-z0-9._-]` characters replaced with `_` before being used as the Keyline identity.

**LDAP injection protection:** Username passed to `ldap.EscapeFilter()` before use in search filter.

---

### OIDC Provider (any compliant IdP)

**Purpose:** Browser-based SSO via OAuth2 authorization code flow with PKCE.

**Library:** Standard `net/http` for HTTP calls; `gopkg.in/square/go-jose.v2` v2.6.0 for JWT validation

**Implementation:** `internal/auth/oidc.go` — `OIDCProvider`

**Protocol flow:**
1. Discovery: `GET {issuer}/.well-known/openid-configuration` (3 retries, exponential backoff)
2. JWKS fetch and cache (24h TTL, background refresh goroutine)
3. Authorization redirect (PKCE S256, state token stored in cache)
4. Token exchange: `POST {token_endpoint}` with `code` + `code_verifier`
5. ID token validation: signature (JWKS), issuer, audience, expiry

**Config keys (all in `oidc:` section of YAML):**
| Key | Env var pattern | Notes |
|-----|----------------|-------|
| `issuer_url` | `${OIDC_ISSUER_URL}` | Must be HTTPS (localhost HTTP allowed for dev) |
| `client_id` | `${OIDC_CLIENT_ID}` | |
| `client_secret` | `${OIDC_CLIENT_SECRET}` | |
| `redirect_url` | — | Must match IdP configuration |
| `scopes` | — | e.g. `[openid, email, profile]` |
| `user_identity_claim` | — | Claim used as username; default `email` |

**Supported IdPs:** Any OIDC-compliant provider (Google, Keycloak, Okta, Azure AD, etc.) via standard discovery endpoint.

**Session storage:** After successful callback, session is stored in cache (`internal/session/`) with configurable TTL. Session cookie: `HttpOnly`, `SameSite=Lax`, `Secure=true` (false for localhost HTTP).

---

## Elasticsearch

**Purpose:** Dynamic user management — create/update ES native users via the Security API so each authenticated user gets a unique ES identity with mapped roles.

**Client:** Standard `net/http` (no ES SDK) — custom client in `internal/elasticsearch/client.go`

**API endpoints used:**
| Operation | HTTP | Path |
|-----------|------|------|
| Create/update user | `PUT` | `/_security/user/{username}` |
| Get user | `GET` | `/_security/user/{username}` |
| Delete user | `DELETE` | `/_security/user/{username}` |
| Validate connection | `GET` | `/_security/_authenticate` |

**Auth:** HTTP Basic Auth using admin credentials (must have `manage_security` privilege)

**Config keys (in `elasticsearch:` section):**
| Key | Env var pattern | Notes |
|-----|----------------|-------|
| `url` | `${ES_URL}` | e.g. `https://elasticsearch:9200` |
| `admin_user` | `${ES_ADMIN_USER}` | Needs `manage_security` privilege |
| `admin_password` | `${ES_ADMIN_PASSWORD}` | |
| `timeout` | — | Default: `30s` |
| `insecure_skip_verify` | — | TLS cert skip; production: `false` |

**Retry logic:** Up to 3 attempts with exponential backoff (1s, 2s, 4s). No retry on `401`/`403`.

**Circuit breaker:** `internal/elasticsearch/circuit_breaker.go` — wraps client to prevent cascade failures.

**OTel tracing:** All ES API calls emit spans under tracer `"keyline"` with span names `elasticsearch.create_or_update_user`, `elasticsearch.get_user`, `elasticsearch.delete_user`, `elasticsearch.validate_connection`.

---

## Cache / Session Store

**Purpose:** Store OIDC state tokens, JWKS cache, user sessions, and encrypted ES credentials.

**Library:** `github.com/wasilak/cachego` v0.0.11 — unified interface over Redis or in-memory backends

**Implementation:** `internal/cache/cache.go` — `InitCache()`; `internal/session/` — session CRUD

**Backends:**

| Backend | Library | Use case |
|---------|---------|----------|
| `redis` | `github.com/redis/go-redis/v9` v9.18.0 | Production; session persistence, horizontal scaling |
| `memory` | `github.com/dgraph-io/badger/v4` or `github.com/patrickmn/go-cache` | Development/testing; single-node only |

**Config keys (in `cache:` section):**
| Key | Env var pattern | Notes |
|-----|----------------|-------|
| `backend` | — | `redis` or `memory` |
| `redis_url` | `${REDIS_URL}` | `redis://[:password@]host[:port][/db]` |
| `redis_password` | `${REDIS_PASSWORD}` | Optional; can be embedded in URL |
| `redis_db` | — | Integer 0–15 |
| `credential_ttl` | — | ES credential cache TTL (e.g. `1h`) |
| `encryption_key` | `${CACHE_ENCRYPTION_KEY}` | Exactly 32 bytes; AES-256-GCM for cached ES passwords |

**Startup validation:** Cache connection is tested at startup (set + get a health-check key). Failure prevents server from starting.

**Credential encryption:** When `user_management.enabled: true`, generated ES passwords are AES-256-GCM encrypted before storing in cache. All Keyline instances sharing a Redis cluster must use the same `encryption_key`.

---

## OpenTelemetry Collector

**Purpose:** Distributed tracing export (OTLP over HTTP/gRPC).

**Library:** `github.com/wasilak/otelgo` v1.3.0 (wraps OpenTelemetry SDK); full OTel SDK v1.42.0

**Implementation:** `internal/observability/tracing.go` — `InitTracer()`

**Config keys (in `observability:` section):**
| Key | Env var / override | Notes |
|-----|-------------------|-------|
| `otel_enabled` | — | `true`/`false` |
| `otel_endpoint` | `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | e.g. `http://otel-collector:4318` |
| `otel_service_name` | `OTEL_SERVICE_NAME` | Default: `keyline` |
| `otel_service_version` | `OTEL_SERVICE_VERSION` | |
| `otel_environment` | `OTEL_DEPLOYMENT_ENVIRONMENT` | |
| `otel_trace_ratio` | — | 0.0–1.0 sampling ratio |

**Propagation:** W3C Trace Context + Baggage

**Fallback:** If OTLP export fails, a no-op tracer is used — server continues running.

---

## Prometheus Metrics

**Purpose:** Expose `/metrics` endpoint for Prometheus scraping.

**Library:** `github.com/prometheus/client_golang` v1.23.2

**Implementation:** `internal/observability/metrics.go`, `internal/observability/handler.go`

**Metrics exposed:**
| Metric | Type | Labels |
|--------|------|--------|
| `auth_attempts_total` | Counter | `method`, `result` |
| `auth_request_duration_seconds` | Histogram | `method` |
| `active_sessions` | Gauge | — |
| `session_operations_total` | Counter | `operation` |
| `oidc_provider_requests_total` | Counter | `endpoint`, `result` |
| `upstream_proxy_duration_seconds` | Histogram | — |
| `concurrent_requests` | Gauge | — |
| `errors_total` | Counter | `error_type` |

**Config:** `observability.metrics_enabled: true` to enable endpoint.

---

## Upstream Proxy (Standalone Mode)

**Purpose:** In `standalone` mode, Keyline acts as a full reverse proxy, forwarding authenticated requests to a backend (typically Kibana).

**Implementation:** `internal/transport/standalone.go`

**Config keys (in `upstream:` section):**
| Key | Notes |
|-----|-------|
| `url` | e.g. `http://kibana:5601` |
| `timeout` | Default: `30s` |
| `max_idle_conns` | Default: `100` |

---

## Environment Variables Summary

All secrets must be environment variables referenced as `${VAR_NAME}` in the YAML config.

| Variable | Integration | Required when |
|----------|------------|---------------|
| `LDAP_URL` | LDAP | `ldap.enabled: true` |
| `LDAP_BIND_DN` | LDAP | `ldap.enabled: true` |
| `LDAP_BIND_PASSWORD` | LDAP | `ldap.enabled: true` |
| `LDAP_SEARCH_BASE` | LDAP | `ldap.enabled: true` |
| `LDAP_GROUP_SEARCH_BASE` | LDAP | Optional |
| `OIDC_ISSUER_URL` | OIDC | `oidc.enabled: true` |
| `OIDC_CLIENT_ID` | OIDC | `oidc.enabled: true` |
| `OIDC_CLIENT_SECRET` | OIDC | `oidc.enabled: true` |
| `ES_URL` | Elasticsearch | `user_management.enabled: true` |
| `ES_ADMIN_USER` | Elasticsearch | `user_management.enabled: true` |
| `ES_ADMIN_PASSWORD` | Elasticsearch | `user_management.enabled: true` |
| `REDIS_URL` | Cache | `cache.backend: redis` |
| `REDIS_PASSWORD` | Cache | Optional |
| `CACHE_ENCRYPTION_KEY` | Cache | `user_management.enabled: true` |
| `SESSION_SECRET` | Sessions | Always |
