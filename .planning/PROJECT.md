# Keyline

## What This Is

Keyline is an authentication proxy for Elasticsearch. It sits in front of an ES cluster and authenticates users via OIDC, local Basic Auth, or LDAP, then creates or updates ES-native users with dynamically generated credentials before proxying or forwarding requests. It operates as a forward-auth sidecar (Traefik/Nginx) or a standalone reverse proxy.

## Core Value

Authenticated users get their own Elasticsearch identities automatically — real accountability and auditing without per-user pre-configuration.

## Current State: Post v2.1

v2.1 (Observability & Integration) shipped 2026-05-18. No active milestone. See backlog below for next candidate.

## Requirements

### Validated

- ✓ OIDC authentication via any compliant provider — v1.0
- ✓ Basic Auth with local user config — v1.0
- ✓ Forward-auth mode (Traefik X-Forwarded-*, Nginx X-Original-*) — v1.0
- ✓ Standalone reverse proxy mode with WebSocket support — v1.0
- ✓ Session management (Redis + in-memory backends) — v1.0
- ✓ Prometheus metrics + OpenTelemetry tracing — v1.0
- ✓ Dynamic Elasticsearch user management (UpsertUser, role mapping, credential caching) — v2.0
- ✓ LDAP authentication with TLS modes (ldaps, starttls, plaintext) — v2.0
- ✓ AES-256-GCM encrypted credential caching — v2.0
- ✓ Configurable CORS allowed origins — v2.0
- ✓ Env-var enforcement for all sensitive config fields — v2.0
- ✓ Circuit breaker on Elasticsearch client — v2.0
- ✓ Module name corrected to github.com/wasilak/keyline — v2.0 ship
- ✓ Documentation updated with accurate v2.0 content and wasilak org references — v2.0 ship
- ✓ Expanded Prometheus metrics: ES user management, LDAP ops, circuit breaker state (METRICS-01–05) — v2.1
- ✓ OTel log bridge (opt-in via otel_enabled) + auth engine spans (OTEL-01–03) — v2.1
- ✓ Structured audit log events for all auth decisions with OTel trace correlation (AUDIT-01–02) — v2.1
- ✓ Auth paths reference documentation with manual test commands for all 5 paths (DOC-03) — v2.1
- ✓ Secan integration architecture design: Option C (hybrid forwardAuth + proxy) recommended (SECAN-01) — v2.1

### Backlog (candidates for v2.2+)

- [ ] DEPL-01: Helm chart published to a Helm repository for `helm repo add` + `helm install`
- [ ] DEPL-02: Kubernetes operator for declarative Keyline configuration
- [ ] OBS-01: Grafana dashboard published alongside Keyline docs
- [ ] OBS-02: Alerting runbook for common Keyline failure modes

### Out of Scope

- Admin UI — browser-based management dashboard; defer to v3+
- Multi-cluster routing — single ES cluster target per instance; defer to v3+
- Re-bind failure recovery in LDAP — current behavior is correct per LDAP spec; document-only if needed
- Secan PoC / implementation — design spike only in v2.1; implementation deferred until architecture validated

## Context

- **Upstream**: `github.com/wasilak/keyline` — Piotr contributing to wasilak's project
- **Predecessor**: elastauth + Authelia (two-service chain); Keyline replaces both
- **Go module**: correctly set to `github.com/wasilak/keyline` since v2.0 ship
- **Observability baseline (post v2.1)**: `/metrics` endpoint with 8 existing + 7 new `keyline_` metrics; loggergo + OTel spans across LDAP, cache, ES upsert; structured audit log on every auth decision
- **Docs site**: Docusaurus at `docs/` directory, published to wasilak.github.io/keyline
- **Secan**: Rust+React ES management GUI; Option C hybrid integration architecture designed in v2.1 — implementation deferred

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
| keyline_ prefix for new v2.1 metrics only | Avoid breaking existing Grafana dashboards on upgrade | ✓ Good |
| Secan as design spike only in v2.1 | "Figure out how and if" — validate architecture before implementation | ✓ Good — Option C documented, impl deferred |
| ESAPICallsTotal in internal/elasticsearch package | Resolves circular import from internal/observability | ✓ Good |
| AuthMethod fixed vocabulary (session/basic/ldap/oidc/unknown) | Prevents inconsistent values across code paths | ✓ Good |
| OTel audit correlation guarded by span.IsRecording() | No zero-value trace IDs when OTel is disabled | ✓ Good |
| Secan Option C hybrid: Traefik forwardAuth + Keyline standalone proxy | ForwardAuth for browser→Secan login, proxy for Secan→ES connections | ✓ Good — two-instance limitation documented |

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
*Last updated: 2026-05-18 — Milestone v2.1 complete*
