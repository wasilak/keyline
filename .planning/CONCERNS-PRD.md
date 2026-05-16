# PRD: Keyline Code Quality & Security Remediation

## Overview

This PRD captures all actionable remediation work identified in the codebase concerns audit (`CONCERNS.md`). Items are grouped by area and prioritized by severity. The goal is to bring Keyline to a production-ready security and quality baseline before the LDAP feature is shipped.

---

## 1. Security Fixes (High Priority)

### 1.1 Restrict CORS to Allowed Origins

**Problem:** The HTTP server currently allows wildcard CORS (`*`). Keyline is a forward-auth proxy that forwards Elasticsearch credentials in requests. A wildcard CORS policy means any browser origin can trigger cross-site requests that carry real ES credentials.

**Requirements:**
- Replace the wildcard CORS origin with a configurable allowlist of origins in `config.yaml`.
- Add a `cors.allowed_origins` field to `LDAPConfig` or a top-level `server` config section.
- If `allowed_origins` is empty or not configured, default to rejecting all cross-origin requests (fail-safe).
- Add validation in `internal/config/validator.go` to warn if origins contains `*`.
- Add unit tests for the CORS middleware behavior with allowed and disallowed origins.

**Location:** `internal/server/server.go`, `internal/config/config.go`

---

### 1.2 Enforce TLS Verification on OTel Collector Connection

**Problem:** The OpenTelemetry collector connection is hardcoded to skip TLS verification (`InsecureSkipVerify: true` or equivalent). This exposes telemetry data to MITM attacks and violates security best practices.

**Requirements:**
- Make OTel TLS verification configurable via `observability.tls_skip_verify` (bool, default `false`).
- When `tls_skip_verify` is `false` (default), use proper TLS verification.
- When `tls_skip_verify` is `true`, log a startup warning: `"OTel collector TLS verification is disabled — not recommended for production"`.
- Add the config field to `config.example.yaml` with a comment explaining the risk.

**Location:** `internal/observability/tracing.go` or `internal/transport/`

---

### 1.3 Enforce Env-Var References for All Sensitive Config Fields

**Problem:** Only `ldap.bind_password` enforces the `${ENV_VAR}` reference format. Other sensitive fields — Elasticsearch credentials, OIDC client secret — can be set as plaintext values in `config.yaml`, creating a risk of secrets committed to version control.

**Requirements:**
- Identify all sensitive config fields: ES username/password, OIDC client secret, any API keys.
- Add validator rules in `internal/config/validator.go` to reject plaintext values for these fields and require `${ENV_VAR}` format, consistent with the existing `ldap.bind_password` enforcement.
- Update `config.example.yaml` to use `${VAR}` placeholders for all sensitive fields.
- Add unit tests covering the new validation rules.

**Location:** `internal/config/config.go`, `internal/config/validator.go`

---

### 1.4 Fix Session Deletion — Replace Overwrite Workaround with True Deletion

**Problem:** Session deletion is implemented by overwriting the session with an empty value because the `cachego` library (`v0.0.11`) lacks a `Delete` method. Stale session data may persist and be exploitable if the cache backend retains the empty entry.

**Requirements:**
- Evaluate whether a newer version of `cachego` provides a `Delete` method; upgrade if available.
- If no upgrade path exists, replace `cachego` with an alternative that supports true deletion (e.g., `patrickmn/go-cache` or implement a thin wrapper with delete support).
- Ensure the session store interface has a `Delete(key string) error` method.
- Update all session expiry and logout code paths to call `Delete`.
- Add unit tests verifying that deleted sessions are not retrievable.

**Location:** `internal/session/store.go`, `internal/usermgmt/`

---

## 2. Error Handling Fixes (Medium Priority)

### 2.1 Handle OIDC State Token Cleanup Errors

**Problem:** Errors returned when cleaning up OIDC state tokens are silently discarded. If cleanup fails, the state token remains valid and can be replayed by an attacker to complete an OAuth flow they should not be able to complete.

**Requirements:**
- Log a warning (at minimum) when OIDC state token cleanup fails, including the error and state token identifier.
- If the cleanup failure is non-recoverable (e.g., store unavailable), return a 500 error rather than proceeding with the flow.
- Add unit tests for the error path.

**Location:** `internal/server/callback.go` or OIDC state handler

---

### 2.2 Handle Session Expiry Deletion Errors

**Problem:** Errors from session expiry deletion are swallowed silently, making it impossible to detect or alert on session store failures.

**Requirements:**
- Log errors from session expiry deletion at `WARN` level with context (session ID, error).
- Do not change the deletion behavior (non-fatal) but ensure observability.
- Add unit tests that verify the warning is emitted on error.

**Location:** Session management code

---

### 2.3 Resolve Graceful Shutdown Connection TODOs

**Problem:** There are three open `// TODO: close connection` comments in the graceful shutdown path. Unclosed connections on shutdown can cause goroutine leaks, port exhaustion, or data loss.

**Requirements:**
- Locate and resolve all three TODO comments.
- Implement proper connection closing in the shutdown sequence with a configurable drain timeout.
- Verify the shutdown sequence closes: LDAP connections (if pooled), ES client connections, OTel exporter.
- Add a shutdown integration test or at minimum verify with a manual test that the process exits cleanly.

**Location:** `cmd/keyline/main.go`

---

## 3. Test Coverage (High Priority)

### 3.1 Add Unit Tests for the Authentication Engine

**Problem:** `internal/auth/engine.go` is 428 lines of core authentication dispatch logic with zero direct unit tests. This is the highest-risk untested code in the codebase — bugs here affect every authentication path.

**Requirements:**
- Create `internal/auth/engine_test.go`.
- Mock dependencies: local users map, LDAP provider, OIDC provider, session store.
- Cover all dispatch paths:
  - Valid session cookie → authenticated
  - Basic Auth: local user correct password → authenticated
  - Basic Auth: local user wrong password → 401, no LDAP fallthrough
  - Basic Auth: unknown user, LDAP enabled → LDAP attempted
  - Basic Auth: unknown user, LDAP disabled → OIDC redirect
  - Basic Auth: LDAP auth success → authenticated
  - Basic Auth: LDAP auth failure → 401
  - No auth header → OIDC redirect
- Verify correct HTTP status codes are returned for each path.
- Achieve >80% line coverage on `engine.go`.

**Location:** `internal/auth/engine_test.go` (new file)

---

### 3.2 Add LDAP Integration Tests

**Problem:** The LDAP provider has no integration tests. All existing tests use mocked connections, meaning real LDAP protocol behavior (TLS negotiation, search pagination, server error codes) is untested.

**Requirements:**
- Add integration tests in `integration/ldap_test.go` gated by `//go:build integration`.
- Use a containerized LDAP server (e.g., `glauth` or `osixia/openldap`) via `testcontainers-go` or a docker-compose fixture.
- Cover: successful authentication, wrong password, user not found, group search, TLS connection (`ldaps`).
- Document how to run integration tests in the README or a `docs/testing.md` file.

**Location:** `integration/ldap_test.go` (new file)

---

## 4. Technical Debt (Medium Priority)

### 4.1 Remove Duplicate OTel Initialization Path

**Problem:** An `InitTracer` function exists but is never called. The actual OTel initialization happens via a different path. The dead code creates confusion about which path is authoritative and may cause future bugs if someone calls `InitTracer` thinking it is the correct entry point.

**Requirements:**
- Identify which OTel init path is actually used at startup.
- Either: delete `InitTracer` if it is fully superseded, or wire it in as the single canonical init path and remove the duplicate.
- Add a comment at the canonical init site explaining it is the single entry point.

**Location:** `internal/observability/tracing.go`

---

### 4.2 Fix Inconsistent Test Env Var Cleanup

**Problem:** Some test helpers call `os.Setenv` without a corresponding `defer os.Unsetenv`, which can cause test pollution — a value set in one test leaks into another if tests run in the same process.

**Requirements:**
- Audit all `os.Setenv` calls in `*_test.go` files.
- Add `defer os.Unsetenv("VAR")` immediately after every `os.Setenv` that lacks one.
- Alternatively, use `t.Setenv` (available since Go 1.17) which auto-cleans up after the test.

**Location:** All `*_test.go` files

---

## 5. Dependency Upgrades (High Priority)

### 5.1 Replace Deprecated go-jose.v2 with go-jose/v3

**Problem:** `gopkg.in/square/go-jose.v2` is deprecated and unmaintained. It may contain unpatched security vulnerabilities. OIDC JWT verification depends on this library.

**Requirements:**
- Replace `gopkg.in/square/go-jose.v2` with `github.com/go-jose/go-jose/v3` (the maintained fork) or migrate JWT handling to `golang.org/x/oauth2` if the dependency surface is small.
- Update all import paths.
- Run the full test suite to verify no regressions.
- Verify the OIDC flow end-to-end after the upgrade.

**Location:** `go.mod`, all files importing `go-jose.v2`

---

### 5.2 Fix Invalid Go Version in go.mod

**Problem:** `go.mod` specifies `go 1.26`, which does not exist. This is likely a typo for `1.22` or `1.23`. An invalid version directive may cause unexpected behavior with toolchain selection or third-party tooling.

**Requirements:**
- Correct the `go` directive in `go.mod` to the actual minimum supported Go version (verify from CI workflow or Dockerfile).
- Run `go mod tidy` after correcting.

**Location:** `go.mod`

---

## 6. LDAP-Specific Fixes (Medium Priority — pre-merge)

### 6.1 Warn on Plaintext LDAP Connection

**Problem:** When `tls_mode` is empty or unrecognized, `dialLDAP` silently falls through to a plaintext LDAP connection. Operators may not realize their credentials are being sent in the clear.

**Requirements:**
- In `dialLDAP`, when `tls_mode` is `""` or `"none"`, log a startup warning: `"LDAP TLS mode is 'none' — credentials will be transmitted in plaintext. Use 'ldaps' or 'starttls' in production."`.
- Do not change the behavior (plaintext is still allowed for dev/test), only add the warning.
- Add a unit test verifying the warning is emitted.

**Location:** `internal/auth/ldap.go`

---

### 6.2 Warn When AttributeMapping Overrides Explicit Fields

**Problem:** When both `attribute_mapping` and explicit attribute fields (e.g., `email_attribute`) are set, `AttributeMapping` silently wins without any indication to the operator. This can cause confusing misconfiguration.

**Requirements:**
- In `NewLDAPProvider`, when an `AttributeMapping` key overrides a non-empty explicit field, log a `DEBUG` or `INFO` message: `"attribute_mapping.email overrides email_attribute — using mapped value"`.
- Add a unit test verifying the log message is emitted in this scenario.

**Location:** `internal/auth/ldap.go`

---

## Out of Scope

- Fixing the `wasilak/cachego` pre-1.0 version beyond what is needed for the session deletion fix (Task 1.4).
- Re-bind failure in LDAP (noted in concerns) — the current behavior is correct per the LDAP spec; the note was about documentation, not a bug.
