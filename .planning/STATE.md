---
gsd_state_version: 1.0
milestone: v2.1
milestone_name: observability-and-integration
status: in_progress
last_updated: "2026-05-17T15:00:00.000Z"
last_activity: 2026-05-17
progress:
  total_phases: 5
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

## Current Position

Phase: 03 (metrics-expansion) — NOT STARTED
Status: Milestone started, no phases executed yet
Last activity: 2026-05-17

## Project Reference

See: .planning/PROJECT.md (updated 2026-05-17)

**Core value:** Authenticated users get their own Elasticsearch identities automatically — real accountability and auditing without per-user pre-configuration.
**Current focus:** Phase 03 — metrics-expansion

## Accumulated Context

### Decisions

- Existing metric names kept as-is (no `keyline_` prefix rename) to avoid breaking existing Grafana dashboards on upgrade
- New v2.1 metrics use `keyline_` prefix consistently
- Secan integration scoped to design spike only — no source code changes to Secan or Keyline in Phase 7
- OTel log bridge (OTEL-01) is opt-in, gated by existing `otel_enabled` config flag

### Codebase Baseline (discovered 2026-05-17)

- `/metrics` endpoint already registered: `internal/server/server.go:140`
- 8 metric types already defined: `internal/observability/metrics.go`
- `loggergo` v1.8.2 already in use: `cmd/keyline/main.go`
- `otelgo/tracing` already initialized: `cmd/keyline/main.go`
- `otelecho` middleware already wired: `internal/server/server.go`
- Custom auth span enhancer already adds `auth.method`, `auth.result`, `auth.username`

### Blockers

(none)

### Todos

(none)
