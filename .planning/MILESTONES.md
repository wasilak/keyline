# Milestones

## v2.0 — Ship (Current)

**Status:** In progress
**Started:** 2026-05-17
**Goal:** Fix module identity and update documentation so the v2.0 feature set can be released cleanly.

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
