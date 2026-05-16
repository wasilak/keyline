# Architecture
_Generated: 2026-05-16 | Focus: arch_

## Summary
Keyline is an authentication proxy for Elasticsearch. It sits in front of an ES cluster and authenticates users via OIDC, local Basic Auth, or LDAP, then creates or updates ES-native users with dynamically generated credentials before proxying or authorizing the request. It operates in two deployment modes: standalone reverse proxy or forward-auth sidecar (Traefik/Nginx).

## System Overview

```text
┌──────────────────────────────────────────────────────────────┐
│               HTTP Request (client / reverse proxy)          │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                   Echo HTTP Server                           │
│   `internal/server/server.go`                                │
│                                                              │
│   Middleware chain:                                          │
│   otelecho → RequestTracing → slog-echo → RequestID →        │
│   Recover → CORS → ConcurrentLimiter → BodySizeLimiter       │
│                                                              │
│   Routes:                                                    │
│   GET /healthz          → handleHealth                       │
│   GET /auth/callback    → handleCallback                     │
│   GET|POST /auth/logout → handleLogout                       │
│   GET /metrics          → observability.MetricsHandler       │
│   ANY /*               → transport.Adapter.HandleRequest     │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│              Transport Adapter (mode-specific)               │
├──────────────────────┬───────────────────────────────────────┤
│  ForwardAuthAdapter  │    StandaloneProxyAdapter             │
│  `internal/transport/│    `internal/transport/               │
│   forward_auth.go`   │     standalone.go`                    │
│                      │                                       │
│  Normalizes Traefik  │  Full reverse proxy with httputil.    │
│  X-Forwarded-* or    │  ReverseProxy; handles WebSocket      │
│  Nginx X-Original-*  │  upgrade; retries on 401              │
│  headers, returns    │  with fresh credentials               │
│  200 + Auth header   │                                       │
└──────────┬───────────┴──────────────┬────────────────────────┘
           │                          │
           └──────────────┬───────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                  Authentication Engine                       │
│   `internal/auth/engine.go`                                  │
│                                                              │
│   Precedence order per request:                              │
│   1. Session cookie  → authenticateWithSession               │
│   2. Basic Auth header                                       │
│      a. Local user exists → authenticateWithBasicAuth        │
│      b. LDAP enabled      → authenticateWithLDAP             │
│   3. OIDC (no creds present) → initiateOIDCFlow (redirect)   │
└──────────┬───────────────────────────────────────────────────┘
           │  (all paths call UpsertUser after identity confirmed)
           ▼
┌──────────────────────────────────────────────────────────────┐
│                 User Management                              │
│   `internal/usermgmt/manager.go`                             │
│                                                              │
│   UpsertUser:                                                │
│   1. Cache lookup (key: keyline:user:<name>:password)        │
│   2. TTL threshold check (10% of TTL remaining → regenerate) │
│   3. RoleMapper.MapGroupsToRoles (wildcard pattern match)    │
│   4. ES Security API: CreateOrUpdateUser                     │
│   5. Cache new password (AES-256-GCM encrypted)              │
│                                                              │
│   InvalidateCache + retry on upstream 401                    │
└──────────┬───────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│            Elasticsearch Security API                        │
│   `internal/elasticsearch/client.go`                         │
│   Admin credentials used for _security/user/<name> API       │
└──────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | Key Files |
|-----------|----------------|-----------|
| `main` | Bootstrap, DI wiring, graceful shutdown | `cmd/keyline/main.go` |
| `config` | Load YAML, env var substitution (`${VAR}`), validation | `internal/config/config.go`, `internal/config/loader.go` |
| `server` | Echo setup, middleware chain, route registration | `internal/server/server.go` |
| `transport` | Deployment-mode-specific request handling and proxying | `internal/transport/` |
| `auth.Engine` | Authentication orchestration with precedence logic | `internal/auth/engine.go` |
| `auth.BasicAuthProvider` | bcrypt password validation against local users config | `internal/auth/basic.go` |
| `auth.LDAPProvider` | LDAP bind-and-search authentication with group lookup | `internal/auth/ldap.go` |
| `auth.OIDCProvider` | OIDC discovery, PKCE flow, JWKS validation, session creation | `internal/auth/oidc.go` |
| `usermgmt.Manager` | Dynamic ES user creation, credential caching, role mapping | `internal/usermgmt/manager.go` |
| `usermgmt.RoleMapper` | Group-to-ES-role pattern matching (wildcard support) | `internal/usermgmt/rolemapper.go` |
| `session` | Session storage/retrieval in cache backend | `internal/session/session.go` |
| `state` | OIDC state token storage (CSRF protection) | `internal/state/state.go` |
| `cache` | Cache backend abstraction (memory / Redis via cachego) | `internal/cache/cache.go` |
| `elasticsearch` | ES Security API client with circuit breaker | `internal/elasticsearch/client.go` |
| `observability` | Prometheus metrics, OTel tracing middleware, logging | `internal/observability/` |

## Authentication Flow

### Session (highest precedence)
1. Extract cookie named `cfg.Session.CookieName` from request
2. Retrieve session JSON from cache (`session:<id>`)
3. Check `session.IsExpired()`
4. Call `userManager.UpsertUser` to get/refresh ES credentials
5. Return `ESAuthHeader` (Basic base64(esUser:esPassword))

### Basic Auth (local users)
1. Decode `Authorization: Basic <base64>` header
2. `Engine.hasLocalUser` checks username exists in `cfg.LocalUsers.Users`
3. `BasicAuthProvider.Authenticate`: bcrypt compare (`golang.org/x/crypto/bcrypt`)
4. Call `userManager.UpsertUser` with `Source: "basic_auth"`
5. Return `ESAuthHeader`

### LDAP
Triggered when `Authorization` header is present, basic auth is disabled or username not found locally, and LDAP is enabled.

1. Decode Basic Auth header
2. `LDAPProvider.Authenticate`:
   - Dial LDAP (plain / StartTLS / LDAPS via `tls_mode`)
   - Bind with service account (`bind_dn` / `bind_password`)
   - Search user by `search_filter` (injection-safe via `ldap.EscapeFilter`)
   - Bind as user (password verification)
   - Re-bind as service account
   - Search groups (optional, non-fatal)
   - Check `required_groups` if configured
   - Normalize username (lowercase, safe chars only)
3. Call `userManager.UpsertUser` with `Source: "ldap"` and resolved groups
4. Return `ESAuthHeader`

LDAP `bind_password` **must** be an `${ENV_VAR}` reference — plain values are rejected at startup.

Attribute mapping: explicit fields (`username_attribute`, `email_attribute`, `display_name_attribute`, `group_name_attribute`) OR the `attribute_mapping` map in config. The map takes precedence.

Defaults: `sAMAccountName` / `mail` / `displayName` / `cn`.

### OIDC (lowest precedence — no credentials present)
1. Engine calls `OIDCProvider.Authenticate`: generate state+PKCE, store in cache, return authorization URL
2. Adapter returns 302 redirect to IdP
3. IdP redirects back to `/auth/callback`
4. `server.handleCallback` → `OIDCProvider.CompleteCallback`:
   - Validate state token (CSRF)
   - Exchange code for tokens (with PKCE verifier)
   - Validate ID token (JWKS, issuer, audience, expiry)
   - Create session, set `keyline_session` cookie
5. Redirect client to `state.Token.OriginalURL`
6. Next request hits session auth path

JWKS refreshed in background every 24h via goroutine started in `NewOIDCProvider`.

## Dynamic User Management (UserManager)

Every authenticated identity (regardless of auth method) is mapped to an Elasticsearch native user. The flow:

1. Cache key: `keyline:user:<username>:password`
2. Cache HIT and TTL > 10% remaining: return cached credentials
3. Cache HIT and TTL ≤ 10%: invalidate, regenerate password, call ES `_security/user/<name>` API, re-cache
4. Cache MISS: generate 32-char random password, create/update ES user via API, cache password

Role assignment: `RoleMapper.MapGroupsToRoles` iterates `cfg.RoleMappings`, matches user groups against patterns (exact, prefix `*`, suffix `*`, middle `*`). If no match, falls back to `cfg.DefaultESRoles`. Failure if neither produces roles.

Standalone mode has retry logic: if upstream ES returns 401, invalidate cache, get fresh credentials, retry once (`proxyWithRetry` in `internal/transport/standalone.go`).

## Deployment Modes

### forward_auth
- Keyline sits alongside Traefik/Nginx as an auth subrequest service
- Normalizes `X-Forwarded-*` (Traefik) or `X-Original-*` (Nginx) headers
- Returns `200` with `Authorization: Basic <esCredentials>` header on success
- Returns `302` for OIDC redirect, `401` for auth failure
- Traefik forwards the `Authorization` header to the actual upstream

### standalone
- Keyline acts as a full reverse proxy in front of Elasticsearch
- Authenticates request, then proxies to `cfg.Upstream.URL` via `httputil.ReverseProxy`
- Replaces client's `Authorization` header with ES credentials
- Supports WebSocket upgrades (TCP tunnel via `net.Dial` + `http.Hijacker`)
- Retry on 401 with fresh credentials

## Middleware Chain (order)

```
otelecho (OTel tracing)
  → RequestTracingMiddleware (custom span attributes)
  → slog-echo (structured request logging with trace correlation)
  → RequestID (X-Request-ID header)
  → Recover (panic recovery)
  → CORS (AllowOrigins: *)
  → ConcurrentRequestLimiter (if cfg.Server.MaxConcurrent > 0)
  → RequestBodySizeLimiter (1 MB hard limit)
  → route handler
```

All middleware is defined and ordered in `internal/server/server.go:New`.

## Error Handling

- Auth failures: `EngineResult.StatusCode` set to 401/500; adapters translate to HTTP responses
- Upstream errors in standalone mode: classified by error type (timeout → 504, connection refused → 502, TLS → 502)
- LDAP group search failure is non-fatal — continues with empty groups
- JWKS refresh failure is non-fatal — continues with cached JWKS
- Startup failures (ES connection, encryption key missing, OIDC discovery) exit with code 1

## Architectural Constraints

- **Global state:** `slog.Default()` set in `main`; OTel tracer is package-global; Prometheus counters/gauges are package-level vars in `internal/observability/metrics.go`
- **Threading:** Go standard concurrency; JWKS refresh runs as a detached goroutine; WebSocket tunnels use two goroutines per connection
- **Circular imports:** None detected; dependency direction is `main → server → transport/auth → usermgmt/session/cache → elasticsearch`
- **No LDAP connection pooling:** A new LDAP connection is opened per authentication request and closed with `defer conn.Close()`
- **cachego Delete workaround:** `session.DeleteSession` sets an empty byte slice instead of deleting (cachego has no Delete method)

## Anti-Patterns

### mapper directory is empty
**What happens:** `internal/mapper/` directory exists but contains no Go files.
**Why it's wrong:** Creates confusion about intent and may indicate abandoned or split logic.
**Do this instead:** Remove the directory or add the files that belong there.

### Hardcoded 1 MB body limit
**What happens:** `RequestBodySizeLimiter(1024 * 1024)` is hardcoded in `server.go:70`.
**Why it's wrong:** Elasticsearch bulk requests can be large; limit is not configurable.
**Do this instead:** Expose `server.max_request_body_bytes` in `ServerConfig` and pass it through.

---
*Architecture analysis: 2026-05-16*
