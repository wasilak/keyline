---
phase: 03-metrics-expansion
plan: 01
subsystem: observability
tags: [go, prometheus, metrics, elasticsearch, ldap, circuit-breaker]

requires: []
provides:
  - "ESAPICallsTotal counter for ES user management operations (create, get, delete)"
  - "CircuitBreakerState gauge tracking open/closed/half-open transitions"
  - "LDAPBindAttempts, LDAPSearchDuration, LDAPConnectionErrors LDAP metrics"
  - "docs/observability/metrics.md — full metric reference with PromQL examples"
affects:
  - internal/elasticsearch/client.go
  - internal/elasticsearch/circuit_breaker.go

tech-stack:
  added: []
  patterns:
    - "Named-return + defer pattern for consistent metric increment on all return paths"
    - "Metrics package co-located with subsystem (internal/auth/metrics.go) to avoid circular imports"

key-files:
  created:
    - internal/auth/metrics.go
    - docs/observability/metrics.md
  modified:
    - internal/elasticsearch/client.go
    - internal/elasticsearch/circuit_breaker.go
    - internal/auth/ldap.go

key-decisions:
  - "ESAPICallsTotal moved to internal/elasticsearch package to resolve circular import (was in internal/observability)"
  - "CircuitBreakerState encodes state as int gauge: Closed=0, Open=1, HalfOpen=2"
  - "LDAP metrics defined in internal/auth/metrics.go (subsystem-local) not internal/observability"
  - "All v2.1 metrics use keyline_ prefix; existing 8 metrics left untouched to avoid dashboard breakage"

patterns-established:
  - "Subsystem-local metrics.go for packages with circular import risk"

requirements-completed:
  - METRICS-01
  - METRICS-02
  - METRICS-03
  - METRICS-04
  - METRICS-05

duration: ~45min
completed: 2026-05-17
---

# Phase 03 Plan 01: Metrics Expansion Summary

**Added Prometheus metrics for ES user management (ESAPICallsTotal), circuit breaker state (CircuitBreakerState), and LDAP operations (LDAPBindAttempts, LDAPSearchDuration, LDAPConnectionErrors). Documented all v2.1 metrics in docs/observability/metrics.md.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-05-17
- **Tasks:** 3
- **Files modified:** 3 (+ 2 created)

## Accomplishments

- Resolved pre-existing circular import blocking ESAPICallsTotal from being wired in client.go; moved definition to internal/elasticsearch package
- Wired ESAPICallsTotal via named-return+defer in client.go for create_user, get_user, delete_user
- Wired CircuitBreakerState gauge at all 3 state transitions in circuit_breaker.go
- Created internal/auth/metrics.go with 3 LDAP metric types; wired all into ldap.go
- Created docs/observability/metrics.md with full metric table, PromQL examples, and suggested alerting rules
- go build ./... clean throughout

## Task Commits

1. **Tasks 1–3** — committed `50a2b138`: ES metrics + circuit breaker gauge + LDAP metrics + docs

## Files Created/Modified

- `internal/auth/metrics.go` — LDAPBindAttempts{result}, LDAPSearchDuration, LDAPConnectionErrors definitions
- `internal/elasticsearch/client.go` — ESAPICallsTotal wired via named-return+defer
- `internal/elasticsearch/circuit_breaker.go` — CircuitBreakerState gauge at all state transitions
- `internal/auth/ldap.go` — LDAP metrics wired at dial, bind, and searchUser call sites
- `docs/observability/metrics.md` — complete metric reference

## Decisions Made

- Moved ESAPICallsTotal to `internal/elasticsearch` package (not `internal/observability`) to break the circular import
- CircuitBreakerState int encoding: Closed=0, Open=1, HalfOpen=2 — matches common Prometheus circuit breaker conventions

## Next Phase Readiness

- All 5 METRICS requirements satisfied; docs cover both existing and new metrics
- Phase 04 (OTel depth) can proceed immediately

---
*Phase: 03-metrics-expansion*
*Completed: 2026-05-17*
