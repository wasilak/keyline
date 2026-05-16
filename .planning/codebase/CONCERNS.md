# Concerns & Risks
_Generated: 2026-05-16 | Focus: concerns_

## Summary
Keyline has two high-severity security concerns (wildcard CORS on a credential-forwarding proxy, insecure OTel TLS default), one deprecated critical dependency (`go-jose.v2`), and zero direct tests for the auth engine (428 lines). The LDAP feature branch is structurally sound with injection escaping in place, but has several silent-failure edge cases that need addressing before merge.

---

## 1. Security Concerns

| Severity | Concern | Location |
|---|---|---|
| 🔴 High | Wildcard CORS on proxy that forwards Elasticsearch credentials — any origin can trigger cross-site requests carrying real ES creds | `internal/server/` |
| 🔴 High | OTel collector connection hardcoded to insecure TLS (no verification) | `internal/transport/` or OTel init |
| 🟡 Medium | Most sensitive config fields lack env-var enforcement — only `ldap.bind_password` enforces `${VAR}` substitution; other secrets (ES credentials, OIDC client secret) can be inlined in plaintext | `internal/config/` |
| 🟡 Medium | Session deletion implemented as overwrite-with-empty workaround rather than true deletion — stale session data may persist | `internal/usermgmt/` or session store |

---

## 2. Missing Error Handling

| Severity | Concern | Location |
|---|---|---|
| 🟡 Medium | OIDC state token cleanup errors silently discarded — enables state token replay risk if cleanup fails | OIDC handler |
| 🟡 Medium | Session expiry deletion errors swallowed silently | Session management |
| 🟡 Medium | Three open TODOs for unclosed connections in graceful shutdown | `cmd/keyline/main.go` or server shutdown |

---

## 3. Test Coverage Gaps

| Severity | Concern | Location |
|---|---|---|
| 🔴 High | No integration tests for LDAP at all — the entire LDAP provider is covered only by unit tests with mocks | `integration/` (missing) |
| 🔴 High | `internal/engine/engine.go` (428 lines, core auth dispatch) has no direct unit tests | `internal/engine/` |
| 🟡 Medium | `dialLDAP` TLS branches (`starttls`, `tls`, plaintext) are not individually tested | `internal/auth/ldap.go` |
| 🟡 Medium | `GroupSearchFilter` `{user_dn}` placeholder substitution not validated or tested | `internal/auth/ldap.go` |

---

## 4. Technical Debt

| Severity | Concern | Location |
|---|---|---|
| 🟡 Medium | Three open `// TODO: close connection` comments in graceful shutdown path | Shutdown code |
| 🟡 Medium | Duplicate OTel init paths — `InitTracer` function exists but is never called; actual init happens elsewhere | OTel setup |
| 🟢 Low | `os.Setenv` calls in test helpers not cleaned up with `defer os.Unsetenv` consistently | `*_test.go` files |

---

## 5. LDAP-Specific Risks (feature/ldap-user-mapping)

**What's done correctly:**
- Username escaping via `ldap.EscapeFilter()` — injection protection is in place ✅
- `bind_password` env-var enforcement — secrets not inlineable ✅
- Username normalization (lowercase/trim) before search ✅

| Severity | Risk | Detail |
|---|---|---|
| 🟡 Medium | Plaintext LDAP silently defaults when `tls_mode` is empty — should warn or reject | `internal/auth/ldap.go` |
| 🟡 Medium | Re-bind failure after successful user bind causes false authentication rejection — connection state not reset between bind attempts | `internal/auth/ldap.go` |
| 🟢 Low | `AttributeMapping` silently overrides explicit fields (e.g. `email`) without warning when both are set | `internal/auth/ldap.go` |

---

## 6. Dependency Concerns

| Severity | Concern | Detail |
|---|---|---|
| 🔴 High | `gopkg.in/square/go-jose.v2` is deprecated and unmaintained — should migrate to `go-jose/go-jose/v3` or `golang.org/x/oauth2` JWT handling | `go.mod` |
| 🟡 Medium | `wasilak/cachego v0.0.11` is pre-1.0 with no `Delete` method — session deletion workaround exists because of this limitation | `go.mod` |
| 🟡 Medium | `go 1.26` in `go.mod` — this Go version does not exist yet (latest stable is 1.23.x); likely a typo for `1.22` or `1.23` | `go.mod` |
