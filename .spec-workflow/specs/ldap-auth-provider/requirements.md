# Requirements Document

## Introduction

Add LDAP as a third authentication provider to Keyline, enabling users to authenticate using their Active Directory credentials via HTTP Basic Auth. This allows a single Keyline instance to simultaneously support OIDC (browser/SSO users), LDAP (AD-based programmatic or human users), and local users (service accounts with bcrypt credentials) — all three coexisting without conflict.

## Alignment with Product Vision

Keyline's vision is a robust, flexible authentication proxy that integrates seamlessly into microservice architectures. LDAP support directly extends this by:
- Removing the dependency on Authelia for LDAP/AD-based auth flows
- Enabling single-instance deployments that cover all authentication personas in an organisation
- Supporting the migration path from legacy LDAP stacks to OIDC without a hard cutover

---

## Requirements

### Requirement 1 — LDAP Authentication Provider

**User Story:** As an infrastructure engineer, I want Keyline to authenticate users against an LDAP/Active Directory server, so that AD-managed users can access protected services without requiring an OIDC identity provider.

#### Acceptance Criteria

1. WHEN a request arrives with an `Authorization: Basic` header AND the username does not exist in `local_users` THEN Keyline SHALL attempt to authenticate the user against the configured LDAP server.
2. WHEN LDAP authentication succeeds THEN Keyline SHALL return a 200 response with the `X-Es-Authorization` header populated with the mapped Elasticsearch credentials.
3. WHEN LDAP authentication fails (wrong password, user not found) THEN Keyline SHALL return a 401 response with no redirect.
4. WHEN the LDAP server is unreachable THEN Keyline SHALL return a 500 response and log the connection error.
5. WHEN `ldap.enabled` is `false` THEN Keyline SHALL skip all LDAP logic and the LDAP config block SHALL NOT be validated.

---

### Requirement 2 — Coexistence of All Three Auth Providers

**User Story:** As a platform operator, I want to run a single Keyline instance that handles OIDC, LDAP, and local user authentication simultaneously, so that I do not need separate proxy instances per authentication type.

#### Acceptance Criteria

1. WHEN `local_users`, `ldap`, and `oidc` are all enabled THEN Keyline SHALL handle each auth type without conflict, according to the following precedence:
   - Valid session cookie → session auth
   - `Authorization: Basic` header + username exists in `local_users` → local user auth (no LDAP fallthrough)
   - `Authorization: Basic` header + username NOT in `local_users` → LDAP auth
   - No header, no cookie → OIDC redirect
2. WHEN a username exists in `local_users` AND the password is wrong THEN Keyline SHALL return 401 immediately and SHALL NOT fall through to LDAP.
3. WHEN a username does NOT exist in `local_users` AND LDAP auth fails THEN Keyline SHALL return 401.
4. WHEN only OIDC and LDAP are enabled (no `local_users`) THEN `Authorization: Basic` requests SHALL go directly to LDAP.

---

### Requirement 3 — LDAP Configuration

**User Story:** As a platform operator, I want to configure LDAP connection details, user search, group search, and attribute mappings in the Keyline config file, so that I can adapt Keyline to any LDAP/AD schema.

#### Acceptance Criteria

1. WHEN Keyline starts with `ldap.enabled: true` THEN it SHALL require `url`, `bind_dn`, `bind_password`, `search_base`, and `search_filter` to be set.
2. WHEN `ldap.url` is set THEN it SHALL start with `ldap://` or `ldaps://`, otherwise startup SHALL fail with a descriptive error.
3. WHEN `ldap.search_filter` is set THEN it SHALL contain the `{username}` placeholder, otherwise startup SHALL fail with a descriptive error.
4. WHEN `ldap.tls_mode` is set THEN it SHALL be one of `none`, `ldaps`, or `starttls`, otherwise startup SHALL fail with a descriptive error.
5. WHEN `ldap.group_search_base` and `ldap.group_search_filter` are both omitted THEN Keyline SHALL skip group fetching and authenticate with empty groups.
6. WHEN `ldap.required_groups` is non-empty AND the authenticated user does not belong to any listed group THEN Keyline SHALL return 401.
7. WHEN attribute mapping fields (`username_attribute`, `email_attribute`, `display_name_attribute`, `group_name_attribute`) are omitted THEN Keyline SHALL use Active Directory defaults (`sAMAccountName`, `mail`, `displayName`, `cn`).

---

### Requirement 4 — Security

**User Story:** As a security-conscious operator, I want LDAP authentication to be implemented securely, so that it does not introduce vulnerabilities into the proxy.

#### Acceptance Criteria

1. WHEN constructing an LDAP search filter THEN Keyline SHALL escape the username using `ldap.EscapeFilter` to prevent LDAP injection.
2. WHEN `ldap.bind_password` is set in config THEN it SHALL be sourced from an environment variable reference, never a hardcoded plaintext value.
3. WHEN `ldap.tls_skip_verify: true` is configured THEN Keyline SHALL log a startup warning (not an error) to alert operators.
4. WHEN logging authentication events THEN Keyline SHALL NOT log passwords or raw credentials.

---

### Requirement 5 — Group-to-Role Mapping Integration

**User Story:** As a platform operator, I want LDAP group memberships to flow through the existing role mapping pipeline, so that LDAP users get the same Elasticsearch role assignment logic as OIDC and local users.

#### Acceptance Criteria

1. WHEN LDAP authentication succeeds THEN the user's groups SHALL be passed to `UpsertUser` via `AuthenticatedUser.Groups`.
2. WHEN `UpsertUser` is called with an LDAP user THEN it SHALL apply the existing `RoleMappings` configuration to determine ES roles.
3. WHEN `AuthenticatedUser.Source` is set THEN it SHALL be `"ldap"` for LDAP-authenticated users.

---

## Non-Functional Requirements

### Code Architecture and Modularity
- **Single Responsibility**: The LDAP provider (`ldap.go`) handles only LDAP protocol interaction. Engine wiring, config validation, and group mapping are in separate files.
- **Provider Pattern**: `LDAPProvider` follows the exact same struct + `Authenticate(ctx, *AuthRequest) *AuthResult` contract as `BasicAuthProvider`.
- **No Shared Mutable State**: Each authentication request opens a fresh LDAP connection (no connection pool in v1).
- **Package-level Helpers**: Credential parsing (`extractCredentials`) is a package-level function shared by both `BasicAuthProvider` and `LDAPProvider`.

### Performance
- LDAP connection timeout defaulting to 10 seconds; configurable via `ldap.connection_timeout`.
- Per-request connection overhead is acceptable for v1; connection pooling is out of scope.
- Auth engine overhead must remain under 200ms in the success path (per product SLA).

### Security
- All LDAP filter inputs must be escaped (`ldap.EscapeFilter`).
- `bind_password` sourced from environment variable only.
- TLS (`ldaps` or `starttls`) required for production; `none` acceptable for internal-only networks.
- `tls_skip_verify` triggers a startup warning log.

### Reliability
- If LDAP group search fails, authentication SHALL still succeed with empty groups (non-fatal).
- If LDAP server is down, the error SHALL be logged and a 500 returned — never a hang beyond `connection_timeout`.

### Usability
- Config file example (`config.example.yaml`) must include a fully commented LDAP section covering all options.
- Startup log must reflect whether LDAP is enabled alongside OIDC and local users.
