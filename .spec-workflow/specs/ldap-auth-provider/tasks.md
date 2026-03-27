# Tasks Document

- [x] 1. Add `LDAPConfig` struct and field to `internal/config/config.go`
  - File: `internal/config/config.go`
  - Add `LDAPConfig` struct with all fields (`Enabled`, `URL`, `BindDN`, `BindPassword`, `ConnectionTimeout`, `TLSMode`, `TLSSkipVerify`, `SearchBase`, `SearchFilter`, `GroupSearchBase`, `GroupSearchFilter`, `UsernameAttribute`, `EmailAttribute`, `DisplayNameAttribute`, `GroupNameAttribute`, `RequiredGroups`)
  - Add `LDAP LDAPConfig \`mapstructure:"ldap"\`` field to existing `Config` struct after `LocalUsers` field
  - _Leverage: existing `LocalUsersConfig` and `OIDCConfig` structs in same file as pattern_
  - _Requirements: 3.1, 3.7_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer familiar with Viper/mapstructure config patterns | Task: Add `LDAPConfig` struct and `LDAP` field to `Config` in `internal/config/config.go`, following the exact same `mapstructure` tag convention as `OIDCConfig` and `LocalUsersConfig` already in that file. All fields are defined in design.md Data Models section. | Restrictions: Do not modify any existing struct or field. Only add new code. | _Leverage: `internal/config/config.go` lines 32–71 for OIDCConfig and LocalUsersConfig patterns_ | Success: `go build ./...` passes, new `LDAPConfig` struct is present with all fields, `Config.LDAP` field added_

- [x] 2. Update `internal/config/validator.go` with LDAP validation
  - File: `internal/config/validator.go`
  - Update the "at least one auth method" check to include `|| cfg.LDAP.Enabled`
  - Add `validateLDAP` function that validates: URL presence and `ldap://`/`ldaps://` scheme, BindDN, BindPassword, SearchBase, SearchFilter contains `{username}`, TLSMode is one of `none`/`ldaps`/`starttls`/`""`
  - Call `errors = validateLDAP(cfg, errors)` from the main `Validate` function
  - _Leverage: existing inline OIDC/LocalUsers validation blocks in `validator.go` lines 17–64 as pattern_
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 1.5_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer | Task: Add LDAP config validation to `internal/config/validator.go`. Add a `validateLDAP(cfg *Config, errors []string) []string` function following the error-accumulation pattern of the existing validator. Update the auth-method check at line 17 to include `cfg.LDAP.Enabled`. See design.md `validateLDAP` section for all validation rules. | Restrictions: Do not change any existing validation logic. Only add the new function and update the one auth-method check line. | _Leverage: `internal/config/validator.go` full file_ | Success: `go build ./...` passes, all LDAP validations described in design.md are implemented_

- [ ] 3. Add LDAP validator tests to `internal/config/validator_test.go`
  - File: `internal/config/validator_test.go`
  - Add 10 test cases for LDAP validation following the standard `testing` package pattern (no testify) used in the rest of that file
  - Tests: `TestValidate_LDAPEnabled_Success`, `TestValidate_LDAPMissingURL`, `TestValidate_LDAPInvalidURLScheme`, `TestValidate_LDAPMissingBindDN`, `TestValidate_LDAPMissingBindPassword`, `TestValidate_LDAPMissingSearchBase`, `TestValidate_LDAPMissingSearchFilter`, `TestValidate_LDAPSearchFilterMissingPlaceholder`, `TestValidate_LDAPInvalidTLSMode`, `TestValidate_LDAPDisabled_NoValidation`
  - _Leverage: existing tests in `internal/config/validator_test.go` for pattern (standard `testing`, `strings.Contains` for error message checks)_
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 1.5_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go test developer | Task: Add 10 LDAP validator test cases to `internal/config/validator_test.go`. Follow the exact pattern of existing tests in that file: standard `testing` package (no testify), `t.Errorf`, `strings.Contains(err.Error(), "...")` for message checks. Test cases are listed in design.md Testing Strategy section. | Restrictions: Use standard `testing` only — no testify. Match existing test function style exactly. | _Leverage: `internal/config/validator_test.go` lines 53–74 for error case pattern_ | Success: `go test ./internal/config/...` passes with all 10 new tests green_

- [ ] 4. Add `github.com/go-ldap/ldap/v3` dependency
  - File: `go.mod`, `go.sum`
  - Run `go get github.com/go-ldap/ldap/v3@v3.4.8`
  - _Requirements: 1.1_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer | Task: Add the go-ldap/ldap/v3 dependency at v3.4.8 by running `go get github.com/go-ldap/ldap/v3@v3.4.8` in the project root. | Restrictions: Do not manually edit go.mod or go.sum. Use `go get` only. | Success: `go.mod` contains `github.com/go-ldap/ldap/v3 v3.4.8`, `go build ./...` passes_

- [ ] 5. Promote `extractCredentials` to package-level function in `internal/auth/basic.go`
  - File: `internal/auth/basic.go`
  - Change `(p *BasicAuthProvider) extractCredentials(header string) (string, string, error)` to `extractCredentials(header string) (string, string, error)` (remove receiver)
  - Update the one call site inside `BasicAuthProvider.Authenticate` to call `extractCredentials(...)` directly
  - _Leverage: `internal/auth/basic.go` — find the method and its single call site_
  - _Requirements: Non-functional — Code Architecture (package-level helpers)_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer | Task: In `internal/auth/basic.go`, promote `extractCredentials` from a method on `*BasicAuthProvider` to a package-level function by removing the receiver. Update the single call site inside `Authenticate` to call `extractCredentials(header)` instead of `p.extractCredentials(header)`. No other changes. | Restrictions: Do not change the function signature or logic, only remove the receiver. Do not change any other file in this task. | _Leverage: `internal/auth/basic.go` — find `extractCredentials` and its call site_ | Success: `go test ./internal/auth/...` passes (all existing basic_test.go tests still green), `extractCredentials` is a package-level function_

- [ ] 6. Implement `LDAPProvider` in `internal/auth/ldap.go`
  - File: `internal/auth/ldap.go` (new file)
  - Implement: `LDAPProvider` struct, `NewLDAPProvider`, `Authenticate`, `connect`, `searchUser`, `searchGroups`, `hasAnyGroup`
  - `Authenticate` flow: `extractCredentials` → `ldap.EscapeFilter` → `connect` → service bind → `searchUser` → user bind → re-bind as service account → `searchGroups` → `required_groups` check → return `AuthResult`
  - Set defaults in `NewLDAPProvider`: `ConnectionTimeout=10s`, `UsernameAttribute="sAMAccountName"`, `EmailAttribute="mail"`, `DisplayNameAttribute="displayName"`, `GroupNameAttribute="cn"`
  - Group search is skipped (returns `[]string{}`) when either `GroupSearchBase` or `GroupSearchFilter` is empty
  - Group search failure is non-fatal: log warning, continue with empty groups
  - Log startup warning if `TLSSkipVerify: true`
  - _Leverage: `internal/auth/basic.go` for provider pattern; design.md for full method signatures and logic; `extractCredentials` (package-level after task 5)_
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.5, 3.6, 3.7, 4.1, 4.3, 4.4, 5.1, 5.3_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer with LDAP/AD experience | Task: Create `internal/auth/ldap.go` implementing the LDAP authentication provider. Follow the exact structure and style of `internal/auth/basic.go`. Full method signatures, logic, and error handling are documented in design.md Components, Architecture, and Error Handling sections. Use `extractCredentials` (package-level, available after task 5). Package is `package auth`. | Restrictions: No connection pooling. One connection per request. Do not use any interface wrapper for ldap.Conn (that is for tests in task 7 only). Use `slog.InfoContext`/`WarnContext`/`ErrorContext` for logging, matching engine.go style. | _Leverage: `internal/auth/basic.go`, design.md full provider spec, `github.com/go-ldap/ldap/v3` docs_ | Success: `go build ./...` passes, all logic from design.md implemented_

- [ ] 7. Wire LDAP provider into auth engine (`internal/auth/engine.go`)
  - File: `internal/auth/engine.go`
  - Add `ldapProvider *LDAPProvider` and `ldapEnabled bool` to `Engine` struct
  - Initialize LDAP provider in `NewEngine` if `cfg.LDAP.Enabled` (mirror basic provider init pattern)
  - Add `hasLocalUser(username string) bool` helper method on `Engine` — linear scan of `e.config.LocalUsers.Users`
  - Update `Authenticate()` Basic Auth block: if `basicEnabled && hasLocalUser(username)` → call `authenticateWithBasicAuth`; else if `ldapEnabled` → call `authenticateWithLDAP`
  - Add `authenticateWithLDAP(ctx, *EngineRequest) *EngineResult` method — mirrors `authenticateWithBasicAuth` exactly, using `ldapProvider.Authenticate` and setting `Source: "ldap"` on `AuthenticatedUser`
  - `extractUsernameFromBasicAuth` helper needed in `Authenticate` to extract username before calling `hasLocalUser` — use the package-level `extractCredentials`, ignore password, ignore error (return `""` on error which means `hasLocalUser` returns false → falls through to LDAP)
  - _Leverage: `internal/auth/engine.go` full file; `internal/auth/basic.go` for `authenticateWithBasicAuth` as mirror pattern_
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 1.2, 1.3, 5.1, 5.2, 5.3_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer | Task: Update `internal/auth/engine.go` to wire in the LDAP provider following design.md Engine updates section and the auth flow diagram. The key design decision (Option B) is: do NOT modify `authenticateWithBasicAuth` — instead add `hasLocalUser` to gate whether basic auth is called, and fall through to LDAP if the username is not locally known. Full details in design.md. | Restrictions: Do NOT modify `authenticateWithBasicAuth`. Do NOT change session or OIDC logic. `authenticateWithLDAP` must be a new separate method mirroring `authenticateWithBasicAuth` structure. | _Leverage: `internal/auth/engine.go` full file, design.md Architecture section_ | Success: `go test ./internal/auth/...` passes, auth flow matches the flowchart in design.md_

- [ ] 8. Add LDAP provider unit tests in `internal/auth/ldap_test.go`
  - File: `internal/auth/ldap_test.go` (new file)
  - Package: `package auth`
  - Framework: `testify` (`require` + `assert`), naming convention `TestLDAPProvider_MethodName_Scenario`
  - Mocking strategy: define a `ldapConn` interface (in test file or `ldap.go`) with `Bind`, `Search`, `Close`, `SetTimeout`; inject mock via a `dialFunc` field or test constructor
  - 11 test cases as listed in design.md Testing Strategy section
  - _Leverage: `internal/auth/basic_test.go` for full test style, setup patterns, and naming conventions_
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 3.6, 4.1_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go test developer with LDAP mocking experience | Task: Create `internal/auth/ldap_test.go` with 11 test cases for `LDAPProvider`. Use testify (require + assert), package `auth`, naming `TestLDAPProvider_MethodName_Scenario`. Design a `ldapConn` interface and mock to avoid real LDAP connections in unit tests. All test cases are listed in design.md Testing Strategy section. Follow `basic_test.go` exactly for style. | Restrictions: No real LDAP connections. All tests must be runnable with `go test` alone (no Docker). Keep mock implementation simple — only what each test needs. | _Leverage: `internal/auth/basic_test.go` for full style reference_ | Success: `go test ./internal/auth/...` passes with all 11 new tests green_

- [ ] 9. Add LDAP section to `config/config.example.yaml`
  - File: `config/config.example.yaml`
  - Add a fully commented LDAP config block after the `local_users` section
  - Include all fields, inline comments explaining each option, examples for both standard and recursive AD membership filter
  - _Leverage: existing `local_users` section in `config/config.example.yaml` for formatting style; design.md `LDAPConfig` data model for all fields_
  - _Requirements: Non-functional — Usability_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Technical writer / Go developer | Task: Add a LDAP configuration block to `config/config.example.yaml` after the `local_users` section. Include all fields from `LDAPConfig`, with inline YAML comments. Show both standard `(member={user_dn})` and recursive AD `(member:1.2.840.113556.1.4.1941:={user_dn})` group filter examples. Follow the style and comment density of the existing `local_users` and `oidc` sections. | Restrictions: Do not change any existing content. Only append/insert the new section. | _Leverage: `config/config.example.yaml` full file for style_ | Success: YAML file is valid, all LDAPConfig fields documented, both group filter variants shown_

- [ ] 10. Add LDAP to startup log in `cmd/keyline/main.go`
  - File: `cmd/keyline/main.go`
  - Find the existing `"Configuration loaded"` log line (around line 97) and add `slog.Bool("ldap_enabled", cfg.LDAP.Enabled)` to it
  - _Leverage: `cmd/keyline/main.go` lines 97–103_
  - _Requirements: Non-functional — Usability_
  - _Prompt: Implement the task for spec ldap-auth-provider, first run spec-workflow-guide to get the workflow guide then implement the task: Role: Go developer | Task: In `cmd/keyline/main.go`, find the `logger.Info("Configuration loaded", ...)` call and add `slog.Bool("ldap_enabled", cfg.LDAP.Enabled)` as an additional slog attribute. One-line change. | Restrictions: Do not change anything else in this file. | _Leverage: `cmd/keyline/main.go` around line 97_ | Success: `go build ./...` passes, the log line includes `ldap_enabled`_
