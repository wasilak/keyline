---
phase: 06-auth-paths-docs
plan: 01
subsystem: docs
tags: [documentation, auth, curl, forward-auth, standalone]

requires:
  - "Audit log format defined (Phase 05)"
provides:
  - "docs/auth-paths.md — complete reference for all 5 auth paths"
  - "curl/http examples for each auth method with expected headers and responses"
  - "Auth method precedence table"
  - "Deployment-mode diagrams (forward-auth vs standalone)"
  - "Expected audit log output examples (success + failure per path)"
affects: []

tech-stack:
  added: []
  patterns: []

key-files:
  created:
    - docs/auth-paths.md
  modified: []

key-decisions:
  - "Covered all 5 auth paths: session, basic, ldap, oidc, unknown/fallthrough"
  - "Included both forward-auth and standalone topology diagrams"
  - "Audit log samples in each section show actual slog field names from Phase 05 implementation"

patterns-established: []

requirements-completed:
  - DOC-03

duration: ~30min
completed: 2026-05-17
---

# Phase 06 Plan 01: Auth Paths Documentation Summary

**Created docs/auth-paths.md covering all 5 auth methods with curl examples, precedence table, deployment diagrams, and audit log output samples.**

## Performance

- **Duration:** ~30 min
- **Completed:** 2026-05-17
- **Tasks:** 1
- **Files modified:** 0 (1 created)

## Accomplishments

- docs/auth-paths.md created with:
  - Auth method precedence table (evaluation order)
  - Deployment diagrams for forward-auth and standalone modes
  - Per-path sections: session, basic, ldap, oidc, unknown
  - curl examples for each path with expected request headers and response codes
  - Expected audit log output (success and failure) for each path, using Phase 05 field names
- go build ./... unaffected (docs-only change)

## Task Commits

1. **Task 1** — committed `6480f02e`: docs/auth-paths.md

## Files Created/Modified

- `docs/auth-paths.md` — complete auth path reference (DOC-03)

## Decisions Made

- Audit log samples in docs use exact field names from logAuditEvent (Phase 05) for accuracy
- Both deployment modes documented (forward-auth and standalone) — users run Keyline in either topology

## Next Phase Readiness

- DOC-03 satisfied
- Phase 07 (Secan spike) can proceed immediately

---
*Phase: 06-auth-paths-docs*
*Completed: 2026-05-17*
