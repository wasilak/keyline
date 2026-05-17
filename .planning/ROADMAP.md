# Roadmap: Keyline v2.0

**Milestone:** v2.0 — Ship
**Phases:** 2
**Requirements:** 4 (all covered)

## Phase Overview

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | Module Rename | 1/1 | Complete   | 2026-05-17 |
| 2 | Documentation Update | 4/4 | Complete   | 2026-05-17 |

---

## Phase 1: Module Rename

**Goal:** Correct the Go module identity throughout the codebase so the project builds correctly under `github.com/wasilak/keyline`.

**Requirements:** MOD-01, MOD-02

**Plans:** 1/1 plans complete

Plans:
- [x] 01-01-PLAN.md — Bulk-replace module path in go.mod + all Go source files, verify build and test gates

**Tasks:**
1. Update `go.mod` module directive from `github.com/yourusername/keyline` to `github.com/wasilak/keyline`
2. Update `go.mod` Go version from `1.26` to a valid version (verify minimum from CI workflow / Dockerfile)
3. Update all `import` statements in `.go` files that reference `github.com/yourusername/keyline/...`
4. Run `go mod tidy` and `go build ./...` to verify no broken imports
5. Run `go test ./...` to confirm tests pass

**Success criteria:**
1. `go build ./...` completes without errors
2. `go test ./...` passes (no broken imports, no test regressions)
3. No remaining references to `yourusername` in Go source files or `go.mod`

---

## Phase 2: Documentation Update

**Goal:** Update README, RELEASE-NOTES.md, and the Docusaurus docs site to accurately reflect v2.0 features (LDAP, dynamic user management, role mapping, Redis caching, CORS, circuit breaker) and replace all placeholder org/URL references with correct wasilak links.

**Requirements:** DOC-01, DOC-02

**Plans:** 4/4 plans complete

Plans:
- [x] 02-01-PLAN.md — Root-level docs: expand README v2.0 features inline, fix RELEASE-NOTES placeholders + add LDAP entry, fix docusaurus.config.js bugs
- [x] 02-02-PLAN.md — Docs reference accuracy: configuration.md missing sections (CORS/LDAP/Observability), stale `user_management.enabled` removal across configuration.md, dynamic-user-management.md, quick-start.md
- [x] 02-03-PLAN.md — Authentication docs: update overview.md for LDAP, create new ldap-authentication.md guide
- [x] 02-04-PLAN.md — Deployment docs: docker.md env-var table (LDAP_BIND_PASSWORD) + health-check reconciliation, role-mappings.md claim field tightening

**Success criteria:**
1. No `your-org`, `yourusername`, or `example.com/your-org` strings remain in README.md or RELEASE-NOTES.md
2. README includes at least one sentence describing each major v2.0 feature
3. All GitHub links in RELEASE-NOTES.md resolve to valid wasilak/keyline URLs
4. Config examples in docs match the actual `config.go` struct field names

---

## Coverage Verification

| Requirement | Phase | Covered |
|-------------|-------|---------|
| MOD-01 | Phase 1 | ✓ |
| MOD-02 | Phase 1 | ✓ |
| DOC-01 | Phase 2 | ✓ |
| DOC-02 | Phase 2 | ✓ |

All 4 requirements covered. ✓

---
*Roadmap created: 2026-05-17*
