# Milestones

## v0.2.0 — Observability & Integration (Complete)

**Status:** Complete
**Started:** 2026-05-17
**Shipped:** 2026-05-18
**Goal:** Deepen observability with expanded Prometheus metrics, OTel log bridge, audit logging, auth path documentation, and a Secan integration design spike.

**Delivered:**
- Expanded Prometheus metrics: 7 new `keyline_` metrics covering ES user management, LDAP ops, circuit breaker state, credential cache (Phases 3 — METRICS-01–05)
- OTel log bridge in `cmd/keyline/main.go` via loggergo — opt-in, gated by `otel_enabled` (Phase 4 — OTEL-01)
- Fine-grained OTel spans: LDAP bind/search in `ldap.go`, cache/ES ops in `manager.go` (Phase 4 — OTEL-02)
- OTel tracing and log export configuration documented (Phase 4 — OTEL-03)
- Structured audit log events via `logAuditEvent` on every auth decision, with OTel trace correlation (Phase 5 — AUDIT-01–02)
- Auth paths reference (`docs/auth-paths.md`) with curl examples and audit log samples for all 5 auth methods (Phase 6 — DOC-03)
- Secan integration architecture design: Option C (hybrid Traefik forwardAuth + Keyline proxy) recommended (`docs/integrations/secan.md`) (Phase 7 — SECAN-01)

**Stats:** 7 commits · 34 files · +2573 / -96 lines

**Archive:** `.planning/milestones/v0.2.0-ROADMAP.md`, `.planning/milestones/v0.2.0-REQUIREMENTS.md`

---

## v2.0 — Ship (Complete)

**Status:** Complete
**Started:** 2026-05-17
**Shipped:** 2026-05-17
**Goal:** Fix module identity and update documentation so the v2.0 feature set can be released cleanly.

**Delivered:**
- Go module name corrected to `github.com/wasilak/keyline` (Phase 1)
- README updated for v2.0 features and wasilak org (Phase 2)
- RELEASE-NOTES.md placeholder URLs fixed (Phase 2)
- Config docs and examples updated for all v2.0 fields (Phase 2)
- LDAP authentication documentation added (Phase 2)
- Deployment docs updated with LDAP env-var table (Phase 2)

---

## v1.0 — Foundation (Pre-GSD baseline)

**Status:** Complete
**Shipped:** ~2026-05 (pre-GSD tracking)
**Summary:** Initial Keyline release replacing elastauth + Authelia. OIDC auth, Basic Auth, forward-auth and standalone proxy modes, session management, Prometheus metrics, OTel tracing.

**Security hardening (shipped alongside core):**
- Configurable CORS allowed origins
- OTel TLS verification configurable
- Env-var enforcement for all sensitive config fields
- Session store extended with true Delete support
- go-jose upgraded v2 → v3

**v2.0 features implemented (awaiting release):**
- Dynamic Elasticsearch user management (UpsertUser, role mapping, credential caching)
- LDAP authentication with TLS modes
- AES-256-GCM credential cache encryption
- Circuit breaker on Elasticsearch client
- Comprehensive auth engine unit tests
- LDAP integration tests via testcontainers-go
