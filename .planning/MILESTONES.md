# Milestones

## v2.1 — Observability & Integration (Current)

**Status:** In progress
**Started:** 2026-05-17
**Goal:** Deepen observability with expanded Prometheus metrics, OTel log bridge, audit logging, auth path documentation, and a Secan integration design spike.

**Target features:**
- Expanded Prometheus metrics: ES user management, LDAP ops, circuit breaker state, credential cache
- OTel log bridge via loggergo (OTLP log export when `otel_enabled`)
- Fine-grained OTel spans inside auth engine internals
- Structured audit log events for all auth decisions (no secrets)
- Auth paths reference documentation with manual test commands
- Secan integration architecture design (forward-auth + ES credential flow)

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
