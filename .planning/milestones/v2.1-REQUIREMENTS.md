# Requirements: Keyline

**Defined:** 2026-05-17
**Core Value:** Authenticated users get their own Elasticsearch identities automatically — real accountability and auditing without per-user pre-configuration.

## v2.1 Requirements

Requirements for the v2.1 observability and integration milestone. Each maps to a roadmap phase.

### Metrics

- [x] **METRICS-01**: Prometheus metrics cover ES user management operations — upsert count (success/failure), AES credential cache hit/miss ratio, role mapping applications
- [x] **METRICS-02**: Prometheus metrics cover LDAP operations — bind attempts (success/failure), search duration, connection errors
- [x] **METRICS-03**: Prometheus metrics cover circuit breaker state — ES circuit breaker open/closed/half-open transitions exposed as a gauge
- [x] **METRICS-04**: All metrics added in v2.1 use the `keyline_` namespace prefix (existing metrics kept as-is to avoid breaking dashboards)
- [x] **METRICS-05**: All metrics documented in `docs/` with name, type, labels, and example PromQL queries

### OpenTelemetry

- [x] **OTEL-01**: When OTel is enabled, loggergo emits logs in OTel format (OTLP log export) in addition to stdout JSON — opt-in via existing `otel_enabled` config flag
- [x] **OTEL-02**: Auth engine internals are instrumented with OTel spans: OIDC token exchange, LDAP bind + search, ES UpsertUser, credential cache get/set operations
- [x] **OTEL-03**: OTel tracing and log export configuration documented in `docs/` — exporter endpoint, TLS, sampling, format

### Audit Logging

- [x] **AUDIT-01**: Every auth decision emits a structured slog audit event with: result (success/failure), auth method (oidc/basic/ldap/forwarded), username (redacted format where possible), source IP, HTTP method + path, timestamp — no credentials or secrets in log output
- [x] **AUDIT-02**: When OTel is enabled, audit log events include the active trace ID for correlation

### Documentation

- [x] **DOC-03**: All five auth paths (OIDC, Basic Auth, LDAP, forward-auth, standalone proxy) have manual test references — curl/http commands, expected headers, expected responses

### Secan Integration

- [x] **SECAN-01**: Integration architecture between Secan and Keyline documented — covers: how Secan sits behind Keyline in forward-auth or proxy mode, how Secan's ES cluster connections relate to Keyline's credential management, what config changes Secan would need, and what limitations exist

---

## v2.0 Requirements (Complete)

Requirements for the v2.0 release milestone.

### Module Identity

- [x] **MOD-01**: Developer can build keyline with the correct Go module name (`github.com/wasilak/keyline`) — all import paths updated consistently
- [x] **MOD-02**: `go.mod` specifies a valid Go version that matches the actual minimum required by the codebase

### Documentation

- [x] **DOC-01**: README accurately describes v2.0 features (dynamic user management, LDAP, role mapping, Redis caching)
- [x] **DOC-02**: RELEASE-NOTES.md contains no placeholder URLs or org references (`your-org` → `wasilak`); all links point to correct GitHub locations

---

## Future Requirements

Features deferred beyond v2.1.

### Deployment

- **DEPL-01**: Helm chart published to a Helm repository for `helm repo add` + `helm install`
- **DEPL-02**: Kubernetes operator for declarative Keyline configuration

### Observability

- **OBS-01**: Grafana dashboard published alongside Keyline docs
- **OBS-02**: Alerting runbook for common Keyline failure modes

## Out of Scope

| Feature | Reason |
|---------|--------|
| Admin UI | Browser-based management dashboard; high complexity, not core to auth proxy value |
| Multi-cluster routing | Single ES cluster target per instance; architectural change deferred to v3+ |
| Re-bind failure recovery (LDAP) | Current behavior is correct per LDAP spec; documentation-only if needed |
| Secan PoC / implementation | Secan integration is a design spike only in v2.1; implementation deferred until architecture is validated |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MOD-01 | Phase 1 | Complete |
| MOD-02 | Phase 1 | Complete |
| DOC-01 | Phase 2 | Complete |
| DOC-02 | Phase 2 | Complete |
| METRICS-01 | Phase 3 | Complete |
| METRICS-02 | Phase 3 | Complete |
| METRICS-03 | Phase 3 | Complete |
| METRICS-04 | Phase 3 | Complete |
| METRICS-05 | Phase 3 | Complete |
| OTEL-01 | Phase 4 | Complete |
| OTEL-02 | Phase 4 | Complete |
| OTEL-03 | Phase 4 | Complete |
| AUDIT-01 | Phase 5 | Complete |
| AUDIT-02 | Phase 5 | Complete |
| DOC-03 | Phase 6 | Complete |
| SECAN-01 | Phase 7 | Complete |

**Coverage:**
- v2.1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-17*
*Last updated: 2026-05-18 — v2.1 milestone complete, all 11 requirements verified*
