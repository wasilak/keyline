---
phase: 04-otel-depth
plan: 01
subsystem: observability
tags: [go, opentelemetry, tracing, logging, ldap, usermgmt]

requires:
  - "OTel SDK initialized in main.go (pre-existing from v2.0)"
provides:
  - "OTel log bridge connecting loggergo to OTLP log exporter (opt-in, otel_enabled gate)"
  - "ldap.bind and ldap.search spans with ldap.username attribute in ldap.go"
  - "cache.get, es.create_credentials, es.upsert_user, cache.set spans in manager.go"
  - "docs/observability/tracing.md — full span inventory and trace documentation"
  - "docs/observability/logging.md — log levels, fields, OTel bridge setup"
affects:
  - cmd/keyline/main.go
  - internal/auth/ldap.go
  - internal/usermgmt/manager.go

tech-stack:
  added: []
  patterns:
    - "OTel log bridge via global LoggerProvider — only active when otelInitialized == true"
    - "Inline span creation with error recording pattern for cache/ES operations"

key-files:
  created:
    - docs/observability/tracing.md
    - docs/observability/logging.md
  modified:
    - cmd/keyline/main.go
    - internal/auth/ldap.go
    - internal/usermgmt/manager.go

key-decisions:
  - "OTel log bridge gated on otelInitialized flag — no new config key; reuses existing otel_enabled"
  - "ldap.go spans use ctx threading: searchUser signature extended to accept context.Context"
  - "manager.go spans are inline (no helper wrapper) — keeps diff minimal"

patterns-established:
  - "OTel log bridge initialization pattern in main.go for conditional OTLP log export"

requirements-completed:
  - OTEL-01
  - OTEL-02
  - OTEL-03

duration: ~60min
completed: 2026-05-17
---

# Phase 04 Plan 01: OTel Depth Summary

**Wired OTel log bridge in main.go (opt-in), added LDAP bind/search spans in ldap.go, added cache/ES operation spans in manager.go. Documented full span inventory and log configuration.**

## Performance

- **Duration:** ~60 min
- **Completed:** 2026-05-17
- **Tasks:** 3
- **Files modified:** 3 (+ 2 created)

## Accomplishments

- OTel log bridge connected to loggergo in main.go — only activates when otelInitialized == true (existing flag)
- ldap.go: ldap.bind and ldap.search spans with ldap.username attribute; ctx threaded into searchUser
- manager.go: cache.get, es.create_credentials, es.upsert_user, cache.set inline spans with error recording
- Created docs/observability/tracing.md with full span inventory table and end-to-end trace example
- Created docs/observability/logging.md with log levels, standard fields, OTel bridge setup, structured log examples
- go build ./... clean throughout

## Task Commits

1. **Tasks 1–3** — committed `80e46f28`: OTel log bridge + LDAP spans + cache/ES spans + docs

## Files Created/Modified

- `cmd/keyline/main.go` — OTel log bridge initialization (OTEL-01)
- `internal/auth/ldap.go` — ldap.bind and ldap.search spans (OTEL-02)
- `internal/usermgmt/manager.go` — cache and ES operation spans (OTEL-02)
- `docs/observability/tracing.md` — span inventory, end-to-end trace, exporter config
- `docs/observability/logging.md` — log levels, fields, OTel bridge setup

## Decisions Made

- OTel log bridge reuses existing `otel_enabled` flag — no new config surface
- searchUser in ldap.go required ctx threading to enable LDAP span propagation

## Next Phase Readiness

- All 3 OTEL requirements satisfied; trace and log docs are complete
- Phase 05 (audit logging) can proceed immediately

---
*Phase: 04-otel-depth*
*Completed: 2026-05-17*
