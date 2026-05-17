---
sidebar_label: LDAP Authentication
sidebar_position: 6
---

# LDAP Authentication

## Overview

LDAP / Active Directory authentication allows corporate directory users to authenticate against Keyline using the standard `Authorization: Basic` HTTP header — the same header used by local users. No separate endpoint or client configuration is required.

LDAP coexists with `local_users`: when a Basic Auth request arrives, the auth engine checks local users first. If no local user matches the username, the request falls through to LDAP automatically. Priority order: **Session → Basic Auth header (local users checked first → LDAP fallback) → OIDC redirect**.

Authentication flow:

1. Keyline connects to the LDAP server using a dedicated **service account** (`bind_dn` / `bind_password`).
2. It searches for the user entry matching the provided username (`search_filter`).
3. It **binds as the user** with the supplied password to validate credentials.
4. It re-binds as the service account and optionally fetches group memberships.
5. If `required_groups` is configured, the user must belong to at least one listed group.

## Configuration

```yaml
ldap:
  enabled: true

  # LDAP server URL — must start with ldap:// or ldaps://
  url: ldaps://ad.corp.yourdomain.com:636

  # Service account for user search
  # bind_password MUST be an environment variable reference — see Secure Credential Handling
  bind_dn: CN=keyline-svc,OU=ServiceAccounts,DC=corp,DC=yourdomain,DC=com
  bind_password: ${LDAP_BIND_PASSWORD}

  # Connection timeout (default: 10s)
  connection_timeout: 10s

  # TLS mode: "ldaps", "starttls", or "none" (dev only — never production)
  tls_mode: ldaps

  # Skip TLS certificate verification — development only, never production
  tls_skip_verify: false

  # User search
  search_base: DC=corp,DC=yourdomain,DC=com
  search_filter: "(sAMAccountName={username})"

  # Group search (optional — omit both to skip group lookup)
  group_search_base: OU=Groups,DC=corp,DC=yourdomain,DC=com
  group_search_filter: "(member={user_dn})"

  # Attribute mapping (defaults shown — override only if your schema differs)
  username_attribute: sAMAccountName
  email_attribute: mail
  display_name_attribute: displayName
  group_name_attribute: cn

  # Access control (optional — omit to allow all authenticated users)
  required_groups:
    - keyline-users
```

### Configuration Reference

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `enabled` | No | `false` | Enable LDAP authentication |
| `url` | Yes | — | LDAP server URL (`ldap://` or `ldaps://`) |
| `bind_dn` | Yes | — | Service account distinguished name |
| `bind_password` | Yes | — | Service account password — must be `${ENV_VAR}` |
| `connection_timeout` | No | `10s` | Dial and operation timeout |
| `tls_mode` | No | `none` | TLS mode: `ldaps`, `starttls`, or `none` |
| `tls_skip_verify` | No | `false` | Skip TLS certificate verification (dev only) |
| `search_base` | Yes | — | Base DN for user search (e.g. `DC=corp,DC=yourdomain,DC=com`) |
| `search_filter` | Yes | — | LDAP filter with `{username}` placeholder (e.g. `(sAMAccountName={username})`) |
| `group_search_base` | No | — | Base DN for group search (omit to skip group lookup) |
| `group_search_filter` | No | — | LDAP filter with `{user_dn}` placeholder (e.g. `(member={user_dn})`) |
| `username_attribute` | No | `sAMAccountName` | Attribute to read the Keyline username from |
| `email_attribute` | No | `mail` | Attribute to read the user's email from |
| `display_name_attribute` | No | `displayName` | Attribute to read the display name from |
| `group_name_attribute` | No | `cn` | Attribute to read group names from |
| `attribute_mapping` | No | — | Alternative map-based attribute override (see Attribute Mapping) |
| `required_groups` | No | — | Access gate: user must belong to at least one listed group |

## TLS Modes

| Mode | Port | Use Case | Production Safe? |
|------|------|----------|-----------------|
| `ldaps` | 636 | LDAP over TLS — connection is encrypted from the start | Yes — **recommended for production** |
| `starttls` | 389 | Upgrade a plain connection to TLS before transmitting credentials | Yes — acceptable if `ldaps` is unavailable |
| `none` | 389 | Plaintext — credentials transmitted unencrypted | **No — development only, never production** |

Set `tls_mode: ldaps` and point `url` to port 636 for new deployments. Use `starttls` only when the LDAP server does not expose port 636. Never use `none` in production: user passwords and the service-account `bind_password` would be transmitted in cleartext.

## Attribute Mapping

Keyline reads user attributes from LDAP using the fields `username_attribute`, `email_attribute`, `display_name_attribute`, and `group_name_attribute`. The defaults work for Active Directory without any extra configuration.

For non-standard schemas (e.g. OpenLDAP), you can override individual fields:

```yaml
ldap:
  username_attribute: uid
  email_attribute: mail
  display_name_attribute: cn
```

Or override multiple attributes at once using the `attribute_mapping` map:

```yaml
ldap:
  attribute_mapping:
    username: uid
    email: mail
    displayName: cn
    groupName: cn
```

Recognized keys for `attribute_mapping`: `username`, `email`, `displayName`, `groupName`.

When both a per-field setting (e.g. `username_attribute`) and an `attribute_mapping` key for the same field are present, `attribute_mapping` takes precedence and Keyline logs a warning.

## Group Search

Group lookup is **optional**. Omit `group_search_base` and `group_search_filter` to skip it entirely — authentication will still succeed and the user will receive no groups.

When configured, Keyline substitutes `{user_dn}` with the authenticated user's distinguished name in `group_search_filter`, then reads the group names using `group_name_attribute` (default: `cn`).

Group lookup failure is **non-fatal**: if the group search fails, Keyline logs a warning and continues with an empty group list. Authentication itself is not blocked.

## Required Groups

`required_groups` is an optional access gate. When set, a user must belong to **at least one** of the listed groups to be admitted. Users who authenticate successfully but are not in any required group receive a 401 response.

```yaml
ldap:
  required_groups:
    - keyline-users
    - keyline-admins
```

Leave `required_groups` empty (or omit it) to allow all authenticated LDAP users.

## Active Directory Example

A complete configuration for an Active Directory environment with LDAPS, group-based access control, and local service accounts coexisting:

```yaml
local_users:
  enabled: true
  users:
    - username: ci-pipeline
      password_bcrypt: ${CI_PASSWORD_BCRYPT}
      groups:
        - ci

ldap:
  enabled: true
  url: ldaps://ad.corp.yourdomain.com:636
  bind_dn: CN=keyline-svc,OU=ServiceAccounts,DC=corp,DC=yourdomain,DC=com
  bind_password: ${LDAP_BIND_PASSWORD}
  connection_timeout: 10s
  tls_mode: ldaps
  tls_skip_verify: false
  search_base: DC=corp,DC=yourdomain,DC=com
  search_filter: "(sAMAccountName={username})"
  group_search_base: OU=Groups,DC=corp,DC=yourdomain,DC=com
  group_search_filter: "(member={user_dn})"
  username_attribute: sAMAccountName
  email_attribute: mail
  display_name_attribute: displayName
  group_name_attribute: cn
  required_groups:
    - keyline-users

role_mappings:
  - claim: groups
    pattern: "keyline-admins"
    es_roles:
      - admin
  - claim: groups
    pattern: "keyline-users"
    es_roles:
      - viewer
```

In this configuration the `ci-pipeline` local user is always checked first; all other usernames fall through to Active Directory.

## OpenLDAP Example

A complete configuration for an OpenLDAP server using StartTLS and `uid`-based username lookup with `attribute_mapping`:

```yaml
ldap:
  enabled: true
  url: ldap://ldap.yourdomain.com:389
  bind_dn: cn=keyline-svc,ou=serviceaccounts,dc=yourdomain,dc=com
  bind_password: ${LDAP_BIND_PASSWORD}
  connection_timeout: 10s
  tls_mode: starttls
  tls_skip_verify: false
  search_base: ou=people,dc=yourdomain,dc=com
  search_filter: "(uid={username})"
  group_search_base: ou=groups,dc=yourdomain,dc=com
  group_search_filter: "(member={user_dn})"
  attribute_mapping:
    username: uid
    email: mail
    displayName: cn
    groupName: cn
  required_groups:
    - keyline-users
```

## Username Normalisation

After a successful LDAP authentication, Keyline normalises the username read from the LDAP username attribute before using it as the internal Keyline identity. The normalisation rules are:

1. **Trim** leading and trailing whitespace.
2. **Lowercase** all characters.
3. **Replace** any character outside `a-z`, `0-9`, `.`, `_`, `-` with an underscore.
4. **Collapse** consecutive underscores into one and remove leading/trailing underscores.

Examples:

| LDAP attribute value | Normalised Keyline username |
|---------------------|----------------------------|
| `Alice.Smith` | `alice.smith` |
| `JOHN DOE` | `john_doe` |
| `user@corp.yourdomain.com` | `user_corp.yourdomain.com` |

## Secure Credential Handling

`bind_password` **must** be an environment variable reference in the form `${ENV_VAR}`. Keyline resolves the reference at startup and returns a fatal error if:

- The value is not in `${ENV_VAR}` form (plaintext values are rejected).
- The referenced environment variable is not set or is empty.

```yaml
# Correct — environment variable reference
bind_password: ${LDAP_BIND_PASSWORD}

# Wrong — plaintext value; Keyline will refuse to start
bind_password: "s3cret"
```

Store `LDAP_BIND_PASSWORD` in a secrets manager (Vault, AWS Secrets Manager, Kubernetes Secrets) and inject it at runtime. Never commit plaintext LDAP credentials to your configuration files or version control.

## Troubleshooting

### Connection failed

**Symptom:** `LDAP connection failed` in logs.

**Causes and fixes:**
- Verify `url` is reachable from the Keyline host (`telnet ad.corp.yourdomain.com 636`).
- Check firewall rules between Keyline and the LDAP server.
- For `ldaps`, ensure the server certificate is trusted by the system CA bundle; set `tls_skip_verify: true` temporarily to confirm TLS is the issue (restore to `false` before production use).

### Bind failure

**Symptom:** `LDAP service account bind failed` in logs; Keyline returns 401.

**Causes and fixes:**
- Verify `bind_dn` is the full distinguished name (e.g. `CN=keyline-svc,OU=ServiceAccounts,DC=corp,DC=yourdomain,DC=com`).
- Verify `LDAP_BIND_PASSWORD` environment variable is set and correct.
- Confirm the service account is not locked or expired in the directory.

### User not found

**Symptom:** `LDAP user search failed` or `user not found` in logs; Keyline returns 401.

**Causes and fixes:**
- Verify `search_base` covers the OU where the user lives.
- Check `search_filter` uses the correct attribute (e.g. `sAMAccountName` for AD, `uid` for OpenLDAP).
- Test the filter directly: `ldapsearch -H ldaps://ad.corp.yourdomain.com:636 -D "<bind_dn>" -w "<password>" -b "<search_base>" "(sAMAccountName=testuser)"`.

### TLS verification failure

**Symptom:** `LDAPS dial failed: x509: certificate signed by unknown authority` in logs.

**Causes and fixes:**
- The LDAP server's TLS certificate is not trusted by the system root CA bundle.
- Install the CA certificate on the Keyline host, or use an LDAP server with a certificate signed by a public CA.
- Do **not** set `tls_skip_verify: true` in production — this disables certificate validation entirely.

### User not in required groups

**Symptom:** `LDAP user not in required groups` in logs; Keyline returns 401 even after successful password validation.

**Causes and fixes:**
- Verify the user is a member of at least one group listed in `required_groups`.
- Check `group_search_base` and `group_search_filter` return the expected groups.
- Test the group filter directly with `ldapsearch` using the user's DN as `{user_dn}`.

## Next Steps

- **[Local Users (Basic Auth)](./local-users-basic-auth.md)** - Configure local service accounts that take precedence over LDAP
- **[Role Mappings](../user-management/role-mappings.md)** - Map LDAP groups to Elasticsearch roles
- **[Authentication Overview](./overview.md)** - How all authentication methods interact
