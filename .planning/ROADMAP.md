# Roadmap: Keyline v2.0

**Milestone:** v2.0 — Ship
**Phases:** 2
**Requirements:** 4 (all covered)

## Phase Overview

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | Module Rename | Correct the Go module identity throughout the codebase | MOD-01, MOD-02 | 3 |
| 2 | Documentation Update | Update all docs and release notes for v2.0 accuracy | DOC-01, DOC-02 | 4 |

---

## Phase 1: Module Rename

**Goal:** Correct the Go module identity throughout the codebase so the project builds correctly under `github.com/wasilak/keyline`.

**Requirements:** MOD-01, MOD-02

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

**Goal:** Update README and RELEASE-NOTES.md to accurately reflect v2.0 features and replace all placeholder org/URL references with correct wasilak links.

**Requirements:** DOC-01, DOC-02

**Tasks:**
1. Update README.md: add v2.0 feature descriptions (dynamic user management, LDAP, role mapping, Redis caching, new config sections)
2. Fix all `your-org` / `your-org.com` / `your-org@example.com` references in RELEASE-NOTES.md → `wasilak`
3. Fix GitHub URLs in RELEASE-NOTES.md → `https://github.com/wasilak/keyline`
4. Verify config examples in docs match actual config struct fields
5. Review `docs/` Docusaurus content for any outdated or placeholder content

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
