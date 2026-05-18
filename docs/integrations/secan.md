# Secan Integration Spike

> **Status:** Design spike — v0.2.0. No source code changes to Secan or Keyline in this phase.
> Implementation deferred pending architecture validation.

## What Is Secan?

[Secan](https://github.com/wasilak/secan) is a Rust+React web GUI for managing Elasticsearch clusters — browsing indices, running queries, managing users and roles. It connects directly to an ES cluster using credentials configured at startup.

The integration problem: Secan has no multi-user auth layer of its own. Keyline has exactly that — OIDC, LDAP, Basic auth, and per-user dynamic ES credential generation. Putting them together gives Secan proper auth and per-user ES identity.

---

## Architecture Options

Three topologies were evaluated.

### Option A — Forward-Auth (Auth delegation only)

```
Browser
  │
  ▼
Traefik
  │  forwardAuth → Keyline /auth/verify (200 / 401)
  │
  ▼
Secan  ──────────────────────────────────────────► Elasticsearch
         single shared service account
```

Traefik sits in front of Secan. On every request, Traefik calls Keyline's `/auth/verify` endpoint. Keyline authenticates the user (OIDC / LDAP / Basic) and returns 200 or 401. Secan itself runs with a single fixed ES service account.

**What Secan needs:**
- Elasticsearch endpoint and a service-account credential pair in its config (unchanged from today)
- No changes to Secan source

**What Keyline needs:**
- Normal forward-auth config in Traefik; Keyline already supports this

**Pros:** Zero changes to Secan. Works today.

**Cons:** All users share one ES account. No per-user ES audit trail. If you revoke a user's Keyline access, they lose the Secan UI, but the ES service account credential is still usable directly.

---

### Option B — Proxy + Credential Injection (ES identity delegation)

```
Browser
  │
  ▼
Secan  ──────────────► Keyline (StandaloneProxy)  ──────────────► Elasticsearch
         HTTP to              injects per-user                      receives per-user
         Keyline's            Authorization: Basic                  ES request
         upstream URL         after auth
```

Secan's ES "cluster URL" is repointed to Keyline. Keyline acts as a reverse proxy to the real ES cluster. On each proxied request, Keyline authenticates the user, generates (or retrieves from cache) a per-user ES credential, and injects `Authorization: Basic <user:pass>` toward ES.

**What Secan needs:**
- ES cluster URL changed to point at Keyline's proxy address instead of ES directly
- Secan's own credential config can be left empty or set to a dummy value (Keyline overrides it)

**What Keyline needs:**
- `standalone_proxy.upstream.url` pointed at the real ES cluster
- Auth method configured (OIDC, LDAP, or Basic)
- User management enabled (`usermgmt.enabled: true`) so per-user ES accounts are created on first auth

**Pros:** Full per-user ES identity. Every ES query is auditable to the individual. ES cluster credentials never leave Keyline.

**Cons:** Secan makes many parallel requests per UI interaction (index list, mapping fetch, query execution). Each must pass through Keyline's auth+proxy layer. Adds latency. Secan may also use ES features (WebSocket, bulk API, streaming) that require connection-level auth — these need verification against Keyline's proxy implementation.

**Known gap:** Secan's React frontend makes ES requests from the browser directly (to the configured cluster URL) for some features. If those bypass Keyline, credential injection doesn't apply. This requires Secan source investigation to confirm — deferred to implementation phase.

---

### Option C — Hybrid (Recommended)

```
Browser
  │
  ├── UI requests ──► Traefik ──forwardAuth──► Keyline /auth/verify
  │                      │
  │                      ▼
  │                   Secan (UI + API server)
  │
  └── ES requests ──► Keyline (StandaloneProxy) ──► Elasticsearch
                        injects per-user credential
```

Forward-auth (Option A) handles Secan's own authentication. Keyline proxy (Option B) handles ES connections. The two functions are split across two Keyline instances (or two listen addresses on one instance — pending config support).

**Why this is recommended:**

Option A alone gives auth but no ES identity. Option B alone requires all ES traffic to flow through Keyline — including Secan's internal health checks and background polling, which may not carry user credentials. Option C avoids the polling problem: Traefik-level forward-auth ensures a human user is behind every Secan UI session, and the proxy handles ES calls that do carry user context.

**What Secan needs:**
- No source changes for the forward-auth path (Traefik handles it)
- ES cluster URL pointed at Keyline proxy address for ES calls
- `open` auth mode in Secan (if it has one) or credentials stripped — Keyline owns auth

**What Keyline needs (two roles):**
- Role 1 — forward-auth verifier: standard Keyline with forward-auth responding to Traefik
- Role 2 — ES proxy: `standalone_proxy` mode with `upstream.url` = real ES cluster, `usermgmt.enabled: true`

---

## Recommended Architecture: Option C

**Short version:** Traefik + Keyline forward-auth for Secan login. Keyline standalone proxy for ES connections. Per-user ES identity for every query.

### Full Request Sequence

```
1. User opens Secan in browser
2. Traefik receives request → calls Keyline /auth/verify
3. Keyline authenticates (OIDC redirect or LDAP/Basic check)
4. On success: Traefik forwards request to Secan with X-Forwarded-User header
5. Secan renders UI

6. User runs a query in Secan UI
7. Secan backend sends ES request to configured cluster URL (= Keyline proxy)
8. Keyline proxy receives request, reads X-Forwarded-User (or re-authenticates)
9. Keyline looks up or generates per-user ES credential (via usermgmt)
10. Keyline injects Authorization: Basic <es-user:es-pass> and forwards to real ES
11. ES processes request under the user's own ES account
12. Response flows back: ES → Keyline → Secan → browser
```

### Config Sketch

**Traefik (Docker label example):**
```yaml
traefik.http.middlewares.keyline-auth.forwardauth.address: "http://keyline:8080/auth/verify"
traefik.http.middlewares.keyline-auth.forwardauth.authResponseHeaders: "X-Forwarded-User,X-Auth-Method"
traefik.http.routers.secan.middlewares: "keyline-auth"
```

**Keyline (forward-auth role):**
```yaml
server:
  port: 8080
oidc:
  enabled: true
  # ... provider config
```

**Keyline (ES proxy role):**
```yaml
server:
  port: 8081
standalone_proxy:
  upstream:
    url: "https://my-es-cluster:9200"
usermgmt:
  enabled: true
  # ... ES admin credentials for user provisioning
```

**Secan:**
```yaml
elasticsearch:
  url: "http://keyline-proxy:8081"
  # credentials omitted — Keyline injects them
```

---

## Limitations and Tradeoffs

| Concern | Detail |
|---------|--------|
| Browser-direct ES calls | If Secan's React frontend makes ES calls directly from the browser (not via Secan's backend), those bypass Keyline. Requires Secan source audit. |
| Parallel connection handling | Secan's backend may open many concurrent ES connections per session. Keyline's proxy must handle this; connection pool sizing matters. |
| Two Keyline instances | Option C requires two Keyline configs. A single instance with multiple listeners is not currently supported — deferred feature. |
| ES user provisioning lag | First request per user triggers ES account creation via usermgmt. This adds ~100–500ms to the first query after login. Subsequent requests use the credential cache. |
| ES user cleanup | Keyline does not currently delete ES users when Keyline users are removed. Stale ES accounts accumulate. Lifecycle management is a v2.2+ item. |
| WebSocket / streaming APIs | If Secan uses ES scroll, async search, or subscriptions, these require persistent connections. Keyline's HTTP proxy supports connection upgrades but this path is untested with ES streaming. |
| Secan `open` mode | It is unclear whether Secan has a flag to disable its own auth entirely. If not, a dummy credential or service account is needed for Secan's startup health check. |

---

## What Would Enable Deeper Integration (Future Scope)

The current design is "Keyline in front of Secan." Deeper integration would let Secan query Keyline for credentials directly:

1. **Keyline credential API**: A `/api/v1/credentials` endpoint that returns a short-lived `{username, password}` JSON for the authenticated user. Secan could call this on session start and use the credentials directly rather than routing all ES traffic through Keyline.

2. **gRPC credential stream**: For high-frequency ES operations, a gRPC stream from Secan to Keyline for credential rotation events. Avoids per-request proxy overhead.

3. **Secan plugin / middleware**: A Rust middleware in Secan's Axum stack that calls Keyline for auth + credentials, removing Traefik from the topology entirely.

These require source changes to both Secan and Keyline and are deferred until Option C is validated in production.

---

## References

- Secan source: https://github.com/wasilak/secan
- Keyline forward-auth config: `docs/auth-paths.md` — Forward-Auth section
- Keyline standalone proxy config: `docs/auth-paths.md` — Standalone Proxy section
- Keyline usermgmt metrics: `docs/observability/metrics.md`
- Keyline audit log: `docs/observability/audit.md`

---

*Spike written: 2026-05-17. No code changes. Implementation deferred to v2.2+.*
