# Roadmap: Keyline v2.1

**Milestone:** v2.1 — Observability & Integration
**Phases:** 5 (Phase 3 through Phase 7, continuing from v2.0)
**Requirements:** 11 (all covered)

## Phase Overview

| # | Phase | Goal | Requirements | Status |
|---|-------|------|--------------|--------|
| 3 | Metrics Expansion | Expand Prometheus coverage + docs | METRICS-01–05 | Pending |
| 4 | OTel Depth | Log bridge + auth engine spans + docs | OTEL-01–03 | Pending |
| 5 | Audit Logging | Structured audit events + trace correlation | AUDIT-01–02 | Pending |
| 6 | Auth Paths Documentation | Manual test reference for all 5 auth paths | DOC-03 | Pending |
| 7 | Secan Integration Spike | Architecture design for Secan + Keyline | SECAN-01 | Pending |

---

## Phase 3: Metrics Expansion

**Goal:** Add Prometheus metrics covering the subsystems introduced in v2.0 (ES user management, LDAP, circuit breaker, credential cache) and document all metrics with PromQL examples.

**Requirements:** METRICS-01, METRICS-02, METRICS-03, METRICS-04, METRICS-05

**Baseline (already exists, no changes needed):**
- `auth_attempts_total` — counter with method label
- `auth_duration_seconds` — histogram
- `active_sessions` — gauge
- `session_operations_total` — counter
- `oidc_provider_requests_total` — counter
- `upstream_proxy_duration_seconds` — histogram
- `concurrent_requests` — gauge
- `errors_total` — counter

**New metrics to add (all with `keyline_` prefix):**
- `keyline_es_upsert_total{result="success|failure"}` — counter
- `keyline_credential_cache_hits_total` / `keyline_credential_cache_misses_total` — counters
- `keyline_role_mapping_applications_total{result="success|failure"}` — counter
- `keyline_ldap_bind_attempts_total{result="success|failure"}` — counter
- `keyline_ldap_search_duration_seconds` — histogram
- `keyline_ldap_connection_errors_total` — counter
- `keyline_circuit_breaker_state` — gauge (0=closed, 1=half-open, 2=open)

**Success criteria:**
1. All new metrics appear in `/metrics` output when the relevant code paths execute
2. Existing metric names unchanged (no regression for existing dashboards)
3. `docs/observability/metrics.md` created with: metric name, type, labels, description, example PromQL for each metric

---

## Phase 4: OTel Depth

**Goal:** Enable loggergo's OTel log export when `otel_enabled` is true, and add fine-grained spans inside auth engine internals so traces show exactly where time is spent during authentication.

**Requirements:** OTEL-01, OTEL-02, OTEL-03

**Baseline (already exists):**
- `otelgo/tracing` initialized in `cmd/keyline/main.go`
- `otelecho` middleware wired — HTTP-level spans
- Custom auth span enhancer adds `auth.method`, `auth.result`, `auth.username` to HTTP spans

**New work:**
- **OTEL-01**: When `otel_enabled=true`, initialize loggergo with OTel log bridge (`LogFormatOTel` or equivalent) — logs emitted as OTLP log records in addition to stdout JSON
- **OTEL-02**: Add child spans inside auth engine for:
  - OIDC: `oidc.token_exchange` — token validation duration
  - LDAP: `ldap.bind` — bind attempt, `ldap.search` — user search
  - ES: `es.upsert_user` — UpsertUser call duration, `es.create_credentials` — credential generation
  - Cache: `cache.get`, `cache.set` — credential cache operations
- **OTEL-03**: Add/update `docs/observability/tracing.md` and `docs/observability/logging.md` covering exporter config, sampling, OTel log format

**Success criteria:**
1. When `otel_enabled=true`, a full auth request trace shows child spans for LDAP bind, LDAP search, ES upsert, cache operations
2. When `otel_enabled=true`, slog output is in OTel-compatible JSON format (or dual-output)
3. When `otel_enabled=false`, behavior is identical to current (no OTel overhead)
4. Documentation covers the full OTLP endpoint + TLS configuration

---

## Phase 5: Audit Logging

**Goal:** Emit a structured slog audit event for every auth decision — no credentials, no secrets — suitable for SIEM ingestion and compliance alerting.

**Requirements:** AUDIT-01, AUDIT-02

**Audit event fields (all decisions):**
- `audit: true` — slog key to distinguish audit events from debug logs
- `auth.result`: `"success"` | `"failure"`
- `auth.method`: `"oidc"` | `"basic"` | `"ldap"` | `"forwarded"`
- `auth.username`: authenticated username (no password, no credential values)
- `http.method`, `http.path`, `http.status`
- `network.client_ip`: source IP from `X-Forwarded-For` or remote addr
- `ts`: RFC3339 timestamp (slog default)
- `trace_id`: active OTel trace ID when `otel_enabled=true` (AUDIT-02)

**Non-goals:**
- No credential values, no tokens, no session keys in any audit event
- No query bodies or ES response payloads

**Success criteria:**
1. Every auth path (OIDC, Basic, LDAP, forwarded, failure) emits exactly one audit event per request
2. `grep '"audit":true'` in log output captures all auth decisions and no non-auth events
3. When OTel is enabled, `trace_id` is present and matches the active trace
4. No secret or credential value appears in any audit log line (verified by code review + grep)

---

## Phase 6: Auth Paths Documentation

**Goal:** Write a practical reference documenting all five auth paths — what Keyline does at each step, and how to manually exercise and verify each path end-to-end.

**Requirements:** DOC-03

**Paths to document:**
1. **OIDC** — browser-initiated flow via `/auth/login`, callback, session cookie
2. **Basic Auth** — local user from config, credential validation
3. **LDAP** — bind + group search, role mapping from LDAP groups
4. **Forward-auth** — Traefik `X-Forwarded-*` headers, Nginx `X-Original-*` headers
5. **Standalone proxy** — direct reverse proxy with ES credential injection

**For each path, document:**
- Prerequisites (config snippet)
- Request flow diagram (text-based)
- `curl` or `http` test command
- Expected response headers and status
- What to check in logs to confirm it worked
- Common failure modes

**Success criteria:**
1. Each auth path has a self-contained section a new contributor can follow
2. All `curl` commands are copy-pasteable against a local Docker Compose stack
3. No reference to internal function names — user-facing description only

---

## Phase 7: Secan Integration Spike

**Goal:** Research and document how Secan (Rust+React ES management GUI) and Keyline can work together — covering the auth delegation architecture and ES credential flow.

**Requirements:** SECAN-01

**Scope (design/research only — no implementation):**

**Architecture options to evaluate:**
1. **Forward-auth mode**: Traefik in front of Secan, Traefik calls Keyline `/auth/verify` as forward-auth. Secan runs in `open` mode and trusts forwarded user headers. Keyline handles OIDC/LDAP.
2. **Proxy + header injection**: Secan's ES cluster connections routed through Keyline. Keyline injects dynamic ES credentials per-user. Secan sees ES requests succeed with user-specific permissions.
3. **Hybrid**: Forward-auth for Secan's own auth (option 1) + credential delegation for ES connections (option 2).

**Deliverable:** `docs/integrations/secan.md` covering:
- Recommended architecture (which option, and why)
- Limitations and tradeoffs
- What Secan config changes are needed (`open` mode, ES endpoint repoint)
- What Keyline config changes are needed
- Sequence diagram (text-based) of a full Secan request
- What would need to change in Secan source for deeper integration (gRPC, credential API) — future scope

**Success criteria:**
1. A developer can read `docs/integrations/secan.md` and understand how to run both services together
2. The architecture decision is explicit: which option is recommended and why
3. The doc clearly marks what is current capability vs. future scope
4. No Secan or Keyline source code changes in this phase

---

## Coverage Verification

| Requirement | Phase | Covered |
|-------------|-------|---------|
| METRICS-01 | Phase 3 | ✓ |
| METRICS-02 | Phase 3 | ✓ |
| METRICS-03 | Phase 3 | ✓ |
| METRICS-04 | Phase 3 | ✓ |
| METRICS-05 | Phase 3 | ✓ |
| OTEL-01 | Phase 4 | ✓ |
| OTEL-02 | Phase 4 | ✓ |
| OTEL-03 | Phase 4 | ✓ |
| AUDIT-01 | Phase 5 | ✓ |
| AUDIT-02 | Phase 5 | ✓ |
| DOC-03 | Phase 6 | ✓ |
| SECAN-01 | Phase 7 | ✓ |

All 11 requirements covered. ✓

---
*Roadmap created: 2026-05-17*
