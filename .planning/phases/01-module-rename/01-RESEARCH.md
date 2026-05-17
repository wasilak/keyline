# Phase 1: Module Rename - Research

**Researched:** 2026-05-17
**Domain:** Go module identity / import path rename
**Confidence:** HIGH

## Summary

This phase is a textual find-and-replace of a module path across 30 Go source files plus one `go.mod` directive. There are no architectural changes, no new packages, and no dependency churn — only the first line of `go.mod` and the import strings that reference the old path change.

The task description flagged `go 1.26` as "potentially invalid", but the installed Go toolchain is `go1.26.3` and the Dockerfile uses `golang:1.26-alpine`. The CI workflow also sets `GO_VERSION: '1.26'` and reads the version from `go.mod` via `go-version-file`. The version is valid and must NOT be changed. MOD-02 ("valid Go version") is already satisfied — the planner should record this as a verification finding, not a change task.

The rename is purely mechanical: `github.com/yourusername/keyline` → `github.com/wasilak/keyline`. No runtime state, external service configuration, OS-registered state, or secret names embed the old path. The change is entirely within the source tree. After the rename, `go build ./...` and `go test ./...` are the correct verification gates.

**Primary recommendation:** Use `sed -i` (or equivalent) for a single-pass bulk replacement, then run `go mod tidy && go build ./... && go test ./...` to verify. Do not update the `go` version directive — it is already correct.

## Project Constraints (from CLAUDE.md)

- Use `rg` (ripgrep) instead of `grep`; use `fd` instead of `find`
- No AI attribution in commit messages
- Surgical changes only — touch nothing outside the explicit scope
- Verification required: `go build ./...` and `go test ./...` must pass before done

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MOD-01 | Developer can build keyline with `github.com/wasilak/keyline` — all import paths updated consistently | 30 Go files + go.mod line 1 need the string replaced. Verified via `rg`. |
| MOD-02 | `go.mod` specifies a valid Go version matching the actual minimum required | Go 1.26.3 is installed; Dockerfile uses `golang:1.26-alpine`; CI uses `GO_VERSION: '1.26'`. The directive `go 1.26` is already valid. No change needed. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Module identity | Build toolchain | — | `go.mod` is read by the Go compiler; no runtime component is involved |
| Import resolution | Build toolchain | — | Go resolves imports at compile time from the module path in `go.mod` |

## Standard Stack

No external packages are installed in this phase. The standard Go toolchain (`go`, `go mod tidy`) is the only tool needed. [VERIFIED: local environment — `go version go1.26.3 darwin/arm64`]

### Core

| Tool | Version | Purpose |
|------|---------|---------|
| `go` | 1.26.3 | Compile, test, module management |
| `go mod tidy` | — | Prune/add missing module requirements after path change |

**Installation:** None required — Go 1.26.3 is already installed.

## Package Legitimacy Audit

No new packages are installed in this phase. This section is not applicable.

## Architecture Patterns

### System Architecture Diagram

```
go.mod (module directive: old path)
    |
    v
30 x *.go files (import blocks: old path)
    |
    v
[sed / go rename] bulk find-and-replace
    |
    v
go.mod (module directive: new path)
30 x *.go files (import blocks: new path)
    |
    v
go mod tidy  -->  go build ./...  -->  go test ./...
```

### Recommended Change Sequence

```
1. Update go.mod line 1
2. Bulk-replace imports in all .go files
3. go mod tidy
4. go build ./...
5. go test ./...
```

### Pattern: Bulk Import Path Replacement with sed

**What:** Single-pass in-place string substitution across all `.go` files.
**When to use:** Module path renames where the old and new paths share no ambiguous substrings.

```bash
# Step 1 — update go.mod module directive
sed -i '' 's|github.com/yourusername/keyline|github.com/wasilak/keyline|g' go.mod

# Step 2 — update all .go files
fd -e go -x sed -i '' 's|github.com/yourusername/keyline|github.com/wasilak/keyline|g' {}

# Step 3 — verify no old string remains
rg "yourusername" . --type go
rg "yourusername" go.mod

# Step 4 — module tidy + build + test
go mod tidy
go build ./...
go test ./...
```

**Note:** On Linux, `sed -i` does not require the empty string argument. The `''` is macOS-specific. Since CI runs on `ubuntu-latest` and the developer machine is macOS (`darwin/arm64`), the plan should be aware of this difference or use `perl -pi -e` for portability.

Portable alternative:
```bash
# Portable (works on both macOS and Linux)
fd -e go -x perl -pi -e 's|github\.com/yourusername/keyline|github.com/wasilak/keyline|g' {}
perl -pi -e 's|github\.com/yourusername/keyline|github.com/wasilak/keyline|g' go.mod
```

### Anti-Patterns to Avoid

- **Editing files one-by-one with an editor:** 30 files, 79 occurrences — error-prone; use bulk replacement.
- **Changing the `go` version directive:** `go 1.26` is valid; the CI and Dockerfile already use 1.26. Do not downgrade.
- **Running `go mod tidy` before updating imports:** `tidy` may fail or produce incorrect results if the module directive still references the old path. Always update `go.mod` first.
- **Using `go get` or `go mod edit` for this change:** Those tools manage dependencies, not the module's own identity. Use sed/perl directly.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Bulk rename | Custom script with per-file loops | `fd -e go -x sed` or `perl -pi -e` for a single command |
| Verification | Manual file inspection | `rg "yourusername"` — zero matches confirms completion |

## Runtime State Inventory

**Trigger:** This phase involves a rename. Inventory completed.

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — no databases store the module path string as a key or ID | None |
| Live service config | None — no external services reference `github.com/yourusername/keyline` | None |
| OS-registered state | None — no Task Scheduler, systemd, or launchd entries reference the module path | None |
| Secrets/env vars | None — no secret keys or env vars use the module path | None |
| Build artifacts | `bin/keyline` binary (if built) and Go build cache reference old symbols | Rebuild after rename (`go build ./...`) — no special migration needed |

**Nothing found in any category** — confirmed by source inspection. The module path is used only within `.go` source files and `go.mod`. Go build cache will be invalidated automatically on next build.

## Common Pitfalls

### Pitfall 1: Incomplete replacement due to cached search results
**What goes wrong:** Developer runs replacement, then searches for remaining instances in a stale buffer/IDE cache — misses instances and ships broken imports.
**Why it happens:** IDE indexing lag.
**How to avoid:** Run `rg "yourusername" . --type go` and `rg "yourusername" go.mod` from the command line after replacement. Zero matches = done.
**Warning signs:** `go build ./...` reports `cannot find package` or `unknown import path`.

### Pitfall 2: go.mod version directive incorrectly changed
**What goes wrong:** Developer sees `1.26` and, thinking it is a placeholder, downgrades to `1.22` or `1.23`, causing build failures if the code uses language features from 1.23+.
**Why it happens:** The ROADMAP.md task description says "verify minimum from CI workflow / Dockerfile" but both CI and Dockerfile already use 1.26.
**How to avoid:** Leave `go 1.26` unchanged. The version is valid and intentional. [VERIFIED: `go version go1.26.3 darwin/arm64` locally; `golang:1.26-alpine` in Dockerfile; `GO_VERSION: '1.26'` in CI]
**Warning signs:** Build errors mentioning missing language features (e.g., range-over-int, enhanced for loops, generics enhancements).

### Pitfall 3: go mod tidy changes go.sum unexpectedly
**What goes wrong:** After `go mod tidy`, `go.sum` changes because of indirect dependency adjustments unrelated to the rename.
**Why it happens:** `go mod tidy` is opportunistic — it cleans up any loose ends, not just the rename-related ones.
**How to avoid:** Expected and acceptable. Commit the updated `go.sum` as part of this phase. Review the diff to confirm it only adds/removes checksums for existing dependencies.
**Warning signs:** `go.sum` contains entries for packages not in `go.mod` — this is fine; `tidy` prunes them.

## Code Examples

### Verifying zero remaining references

```bash
# Confirm replacement is complete
rg "yourusername" . --type go
# Expected output: (no matches / exit code 1)

rg "yourusername" /Users/piotrek/git/keyline/go.mod
# Expected output: (no matches / exit code 1)
```

### Confirming the build gate

```bash
go build ./...
# Expected: no output, exit code 0

go test ./...
# Expected: all packages pass
```

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| `gorename` tool for import renames | `sed`/`perl` bulk replace | `gorename` was deprecated; simple string replacement is idiomatic for module path changes |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `go 1.26` in `go.mod` is a valid, intentional version (not a placeholder) | Standard Stack, Pitfall 2 | Low — confirmed by local Go 1.26.3 install, Dockerfile, and CI config; risk is negligible |

**All other claims in this research were verified by direct inspection of the codebase, go.mod, Dockerfile, and CI workflow files.**

## Open Questions

None. All relevant facts were confirmed by direct codebase inspection.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` | Build, test, mod tidy | Yes | 1.26.3 | — |
| `fd` | Bulk file enumeration | Yes (in PATH per CLAUDE.md) | — | `find` (requires explicit permission per CLAUDE.md) |
| `rg` | Verification search | Yes (in PATH per CLAUDE.md) | — | — |
| `sed` / `perl` | In-place file replacement | Yes (macOS system) | — | Either works; `perl -pi -e` is cross-platform |

**Missing dependencies with no fallback:** None.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` package |
| Config file | none (standard `go test`) |
| Quick run command | `go build ./...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MOD-01 | All imports resolve under new module path | build smoke | `go build ./...` | N/A (built-in) |
| MOD-02 | Go version directive is valid | build smoke | `go build ./...` | N/A (built-in) |

### Sampling Rate

- **Per task commit:** `go build ./...`
- **Phase gate:** `go build ./... && go test ./...` green before close

### Wave 0 Gaps

None — existing test infrastructure (`go test ./...`) covers all phase requirements. No new test files needed.

## Security Domain

This phase makes no security-relevant changes. It is a mechanical string replacement of a module path in source files. No authentication, session management, input validation, cryptography, or access control is modified. Security domain is not applicable.

## Sources

### Primary (HIGH confidence)
- Direct inspection of `/Users/piotrek/git/keyline/go.mod` — module path and Go version confirmed
- Direct inspection of `/Users/piotrek/git/keyline/Dockerfile` — `golang:1.26-alpine` confirmed
- Direct inspection of `/Users/piotrek/git/keyline/.github/workflows/ci.yml` — `GO_VERSION: '1.26'` confirmed
- `go version` output on developer machine — `go1.26.3 darwin/arm64` confirmed
- `rg "yourusername"` scan — 30 Go files, 79 occurrences confirmed

### Secondary (MEDIUM confidence)
- Go module documentation (standard `go mod` toolchain behavior)

## Metadata

**Confidence breakdown:**
- Scope of change: HIGH — fully inventoried by direct rg scan
- Go version validity: HIGH — confirmed by three independent sources (local, Dockerfile, CI)
- Replacement strategy: HIGH — standard Go ecosystem practice
- No regressions expected: HIGH — pure textual rename, no logic change

**Research date:** 2026-05-17
**Valid until:** N/A — this is a one-time mechanical change; research does not expire
