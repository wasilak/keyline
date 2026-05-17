# Keyline

## What This Is

Keyline is an authentication proxy for Elasticsearch. It sits in front of an ES cluster and authenticates users via OIDC, local Basic Auth, or LDAP, then creates or updates ES-native users with dynamically generated credentials before proxying or forwarding requests. It operates as a forward-auth sidecar (Traefik/Nginx) or a standalone reverse proxy.

## Core Value

Authenticated users get their own Elasticsearch identities automatically — real accountability and auditing without per-user pre-configuration.

## Current Milestone: v2.1 Observability & Integration

**Goal:** Deepen observability with expanded Prometheus metrics, OTel log bridge, audit logging, auth path documentation, and a Secan integration design spike.

**Target features:**
- Expanded Prometheus metrics (ES user management, LDAP ops, circuit breaker state, credential cache)
- OTel log bridge via loggergo when `otel_enabled`
- Fine-grained OTel spans inside auth engine internals (OIDC token exchange, LDAP, ES upsert, cache)
- Structured audit log events for all auth decisions (no secrets)
- Auth paths reference documentation with manual test commands for all 5 paths
- Secan integration architecture design (forward-auth + ES credential delegation)

## Requirements

### Validated

- ✓ OIDC authentication via any compliant provider — v1.0
- ✓ Basic Auth with local user config — v1.0
- ✓ Forward-auth mode (Traefik X-Forwarded-*, Nginx X-Original-*) — v1.0
- ✓ Standalone reverse proxy mode with WebSocket support — v1.0
- ✓ Session management (Redis + in-memory backends) — v1.0
- ✓ Prometheus metrics + OpenTelemetry tracing — v1.0
- ✓ Dynamic Elasticsearch user management (UpsertUser, role mapping, credential caching) — v2.0 impl
- ✓ LDAP authentication with TLS modes (ldaps, starttls, plaintext) — v2.0 impl
- ✓ AES-256-GCM encrypted credential caching — v2.0 impl
- ✓ Configurable CORS allowed origins — v2.0 impl
- ✓ Env-var enforcement for all sensitive config fields — v2.0 impl
- ✓ Circuit breaker on Elasticsearch client — v2.0 impl
- ✓ Module name corrected to github.com/wasilak/keyline — v2.0 ship
- ✓ Documentation updated with accurate v2.0 content and wasilak org references — v2.0 ship

### Active

- [ ] METRICS-01 through METRICS-05: Expanded Prometheus metrics coverage + docs
- [ ] OTEL-01 through OTEL-03: OTel log bridge, auth engine spans, configuration docs
- [ ] AUDIT-01 through AUDIT-02: Structured audit log events with trace correlation
- [ ] DOC-03: Auth paths reference with manual test commands
- [ ] SECAN-01: Secan integration architecture design

### Out of Scope

- Admin UI — browser-based management dashboard; defer to v3+
- Multi-cluster routing — single ES cluster target per instance; defer to v3+
- Re-bind failure recovery in LDAP — current behavior is correct per LDAP spec; document-only if needed
- Secan PoC / implementation — design spike only in v2.1; implementation deferred until architecture validated

## Context

- **Upstream**: `github.com/wasilak/keyline` — Piotr contributing to wasilak's project
- **Predecessor**: elastauth + Authelia (two-service chain); Keyline replaces both
- **Go module**: correctly set to `github.com/wasilak/keyline` since v2.0 ship
- **Observability baseline**: `/metrics` endpoint, 8 metric types, loggergo + otelgo already integrated
- **Docs site**: Docusaurus at `docs/` directory, published to wasilak.github.io/keyline
- **Secan**: Rust+React ES management GUI with OIDC/local/open auth modes; integration is a v2.1 spike

## Constraints

- **Compatibility**: No breaking changes to config YAML structure between v1 → v2 for basic OIDC/Basic Auth users
- **Metrics compatibility**: Existing metric names kept as-is; `keyline_` prefix only on new v2.1 metrics
- **Go version**: Target Go 1.22+ (testcontainers-go and dependencies require modern Go)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| cachego with extended Delete interface | cachego v0.11 lacks Delete; created extended interface wrapper | ✓ Good |
| go-jose/v3 over golang.org/x/oauth2 JWT | Smaller migration surface, maintained fork of v2 | ✓ Good |
| testcontainers-go for LDAP integration tests | Real protocol behavior untestable with mocks | ✓ Good |
| Circuit breaker on ES client | Prevent cascade failures when ES is down | ✓ Good |
| keyline_ prefix for new v2.1 metrics only | Avoid breaking existing Grafana dashboards on upgrade | — Pending |
| Secan as design spike only in v2.1 | "Figure out how and if" — validate architecture before implementation | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-17 — Milestone v2.1 started*
