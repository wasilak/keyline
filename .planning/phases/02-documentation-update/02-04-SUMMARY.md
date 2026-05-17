---
phase: 02-documentation-update
plan: "04"
subsystem: docs
tags: [docker, ldap, role-mappings, oidc, deployment]

# Dependency graph
requires:
  - phase: 02-documentation-update
    provides: Phase context and pattern map identifying accuracy gaps in docs

provides:
  - docker.md with LDAP env-var table (LDAP_BIND_PASSWORD, LDAP_URL, LDAP_BIND_DN, LDAP_SEARCH_BASE, LDAP_GROUP_SEARCH_BASE)
  - docker.md health check endpoint confirmed correct (/healthz matches server route)
  - role-mappings.md claim field description reflects unrestricted struct definition

affects: [deployment-docs, user-management-docs]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "LDAP env vars documented with ${ENV_VAR} reference pattern matching bind_password enforcement in ldap.go"
    - "claim field described as open-ended with typical values, not enum-constrained"

key-files:
  created: []
  modified:
    - docs/docs/deployment/docker.md
    - docs/docs/user-management/role-mappings.md

key-decisions:
  - "Health check path /healthz confirmed correct — server.go line 131 registers GET /healthz; docker.md was already accurate, no change needed"
  - "Added five LDAP env-var rows to cover LDAP_BIND_PASSWORD (required), LDAP_URL, LDAP_BIND_DN, LDAP_SEARCH_BASE, LDAP_GROUP_SEARCH_BASE — pattern map showed URL/DN-style vars present in existing table (REDIS_URL)"
  - "claim field description changed from implicit enum to open string with typical examples — RoleMapping.Claim is plain string with no constraint in config.go"

patterns-established:
  - "Env-var table rows describe required-when-feature-enabled vars with conditional Required column value"

requirements-completed: [DOC-01]

# Metrics
duration: 2min
completed: 2026-05-17
---

# Phase 2 Plan 4: Deployment and Role-Mappings Doc Accuracy Fixes Summary

**LDAP env-var coverage added to docker.md (5 new rows) and role-mappings.md claim field tightened from implicit enum to open OIDC claim string with typical examples**

## Performance

- **Duration:** 2 min
- **Started:** 2026-05-17T13:13:23Z
- **Completed:** 2026-05-17T13:14:53Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added five LDAP-related env-var rows to docker.md: LDAP_BIND_PASSWORD (primary gap), LDAP_URL, LDAP_BIND_DN, LDAP_SEARCH_BASE, LDAP_GROUP_SEARCH_BASE
- Confirmed health check endpoint `/healthz` in docker.md is correct — server.go line 131 registers `GET /healthz`; no change needed to docker.md (quick-start.md uses wrong `/_health` path, handled by a different plan)
- Confirmed image reference `ghcr.io/wasilak/keyline:latest` is correct; ES version pin 9.3.1 left intentionally unchanged per pattern map
- Changed role-mappings.md claim field description from implied enum (`groups` or `email`) to accurate open string with typical examples, matching `RoleMapping.Claim string` in config.go

## Task Commits

Each task was committed atomically:

1. **Task 1: Update docker.md env-var table and reconcile health check endpoint** - `eb71084` (docs)
2. **Task 2: Tighten role-mappings.md claim field description** - `0f532a3` (docs)

**Plan metadata:** committed in final docs commit

## Files Created/Modified
- `docs/docs/deployment/docker.md` - Added 5 LDAP env-var rows to Environment Variables table
- `docs/docs/user-management/role-mappings.md` - Updated claim field description to reflect unrestricted struct

## Decisions Made
- Health check path `/healthz` confirmed correct via server.go — docker.md needed no change. The discrepancy is in quick-start.md (uses `/_health`), which is a different plan's responsibility.
- Added all five LDAP env-var rows (not just LDAP_BIND_PASSWORD) since the existing table already contained a URL-style row (REDIS_URL) and the pattern map said to add matching rows for URL/DN vars when such patterns exist.
- ldap.go lines 115-128 confirm `ldap.bind_password` MUST be an `${ENV_VAR}` reference — documented LDAP_BIND_PASSWORD accordingly.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All DOC-01 deployment-layer accuracy issues are now resolved
- quick-start.md health endpoint fix (`/_health` → `/healthz`) confirmed outstanding — tracked in another plan
- Phase 2 documentation update plans complete for deployment and role-mappings docs

## Self-Check

- `docs/docs/deployment/docker.md` exists: FOUND
- `docs/docs/user-management/role-mappings.md` exists: FOUND
- Task 1 commit eb71084: verified in git log
- Task 2 commit 0f532a3: verified in git log
- `rg "LDAP_BIND_PASSWORD" docs/docs/deployment/docker.md` — 1 match: PASS
- Health check `/healthz` matches server.go route registration: PASS
- No restricted claim phrasing in role-mappings.md: PASS

## Self-Check: PASSED

---
*Phase: 02-documentation-update*
*Completed: 2026-05-17*
