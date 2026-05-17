# Phase 2: Documentation Update - Pattern Map

**Mapped:** 2026-05-17
**Files analyzed:** 9 documentation files + 1 config JS file
**Analogs found:** 8 / 10 (2 files need new content, not just updates)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `README.md` | doc, root | reference | `docs/docs/getting-started/about.md` | role-match |
| `RELEASE-NOTES.md` | doc, changelog | reference | `docs/docs/changelog.md` | role-match |
| `docs/docs/configuration.md` | doc, reference | reference | `config/config.example.yaml` | exact (source of truth) |
| `docs/docs/authentication/overview.md` | doc, overview | reference | itself (stale) + `internal/auth/engine.go` | partial |
| `docs/docs/authentication/oidc-authentication.md` | doc, guide | reference | itself (stale) | self |
| `docs/docs/authentication/ldap-authentication.md` | doc, guide (NEW) | reference | `docs/docs/authentication/local-users-basic-auth.md` | role-match |
| `docs/docs/user-management/dynamic-user-management.md` | doc, guide | reference | itself + `internal/usermgmt/` | partial |
| `docs/docs/user-management/role-mappings.md` | doc, guide | reference | itself + `config/config.example.yaml` | partial |
| `docs/docs/deployment/docker.md` | doc, guide | reference | itself | self |
| `docs/docusaurus.config.js` | config, JS | N/A | itself | self |

---

## Pattern Assignments

### `README.md` (root doc — expand v2.0 feature sections inline)

**Analog:** `docs/docs/getting-started/about.md` (lines 1-79)

**Current structure** (README.md, 140 lines):
- Headline + tagline (lines 1-7)
- Architecture diagram (lines 9-27)
- "What changed from elastauth" comparison table (lines 29-36)
- Quick Start: Docker + minimal config + run (lines 38-87)
- "Why Keyline?" (For elastauth users + For new users) (lines 89-105)
- Key Features (lines 107-113)
- Documentation links (lines 115-123)
- Development section (lines 125-136)
- License (lines 138-140)

**What needs expanding (D-05, D-06, specifics section):**

1. `server.cors.allowed_origins` — not mentioned in README at all. Source: `config/config.example.yaml` lines 33-43.
2. LDAP auth — README architecture diagram says "OIDC + Basic" only (line 17); the comparison table says "LDAP only (via Authelia)" as the old way, but LDAP is now a built-in v2.0 feature. Needs adding to Key Features and the "What changed" table row.
3. `server.max_concurrent` + circuit breaker pattern — not mentioned. Source: `config/config.example.yaml` lines 28-30.
4. `redirect_url` example uses `https://auth.example.com/auth/callback` — per D-06 replace with `https://auth.yourdomain.com/auth/callback` pattern.
5. Key Features bullet list (lines 107-113) is thin. Missing: LDAP with TLS modes, circuit breaker, CORS, env-var enforcement for sensitive fields.

**Architecture diagram fix** (lines 16-18): The `Auth[Auth Engine<br/>OIDC + Basic]` label should become `Auth[Auth Engine<br/>OIDC + Basic + LDAP]` to reflect v2.0.

**URL pattern to use for all config examples in README:**
```yaml
redirect_url: https://auth.yourdomain.com/auth/callback
```
(Replace existing `https://auth.example.com/auth/callback` on line 59.)

**CORS snippet to add under "Key Features" or "Minimal Configuration":**
```yaml
server:
  cors:
    allowed_origins:
      - "https://kibana.yourdomain.com"
```
Source: `config/config.example.yaml` lines 34-43.

**LDAP snippet to reference in Key Features:**
```yaml
ldap:
  enabled: true
  url: ldaps://ad.corp.yourdomain.com:636
  tls_mode: ldaps  # none | ldaps | starttls
```
Source: `config/config.example.yaml` lines 144-210.

---

### `RELEASE-NOTES.md` (changelog — fix `your-org` placeholder refs)

**Analog:** `docs/docs/getting-started/about.md` (for correct GitHub URLs) + `go.mod` for module identity.

**All `your-org` occurrences** (8 total, found by grep):

| Line | Current | Correct replacement |
|---|---|---|
| 324 | `https://github.com/your-org/keyline/releases/tag/v2.0.0` | `https://github.com/wasilak/keyline/releases/tag/v2.0.0` |
| 334 | `https://github.com/your-org/keyline` | `https://github.com/wasilak/keyline` |
| 335 | `https://github.com/your-org/keyline/tree/main/docs` | `https://github.com/wasilak/keyline/tree/main/docs` |
| 336 | `https://github.com/your-org/keyline/issues` | `https://github.com/wasilak/keyline/issues` |
| 369 | `https://github.com/your-org/keyline/issues` | `https://github.com/wasilak/keyline/issues` |
| 370 | `https://github.com/your-org/keyline/discussions` | `https://github.com/wasilak/keyline/discussions` |
| 371 | `security@your-org.com` | `security@wasilak.github.com` (or omit — no canonical security email exists) |
| 406 | `https://github.com/your-org/keyline/compare/v1.0.0...v2.0.0` | `https://github.com/wasilak/keyline/compare/v1.0.0...v2.0.0` |

**Also fix** (line 319): `docker pull keyline:v2.0.0` — should be `docker pull ghcr.io/wasilak/keyline:v2.0.0` (matches the correct image ref in `docs/docs/deployment/docker.md` line 19).

**D-08 — intentional example.com values to KEEP as-is:**
- Line 110: `email: testuser@example.com`
- Line 309: `pattern: "*@admin.example.com"` (all YAML snippet example values)
- These are config example placeholders, not org references.

**D-07 security email guidance:** No canonical `wasilak` security contact email exists in any other file. Safest replacement: `security@wasilak.github.com` — but planner should note this is a best-effort fix; maintainer may prefer to remove the line entirely.

**Also fix — `LDAP` not in v2.0 new features list (missing feature):**
The v2.0.0 section has no mention of LDAP auth being added in v2.0. This is a major feature gap in the release notes. A short bullet under "New Features" is warranted.
Pattern for the new entry (copy structure from lines 29-32):
```markdown
#### LDAP Authentication (Active Directory / OpenLDAP)
- Three TLS modes: `ldaps`, `starttls`, `none` (plaintext)
- Configurable service account bind, user search, and group search
- Username normalisation (trim, lower-case, unsupported-char removal)
- Optional `required_groups` access gate
- Full integration with dynamic user management and role mappings
```

---

### `docs/docs/configuration.md` (config reference — accuracy review)

**Source of truth:** `internal/config/config.go` (all struct fields) + `config/config.example.yaml`.

**Issues found (cross-reference config.go lines 8-184 against configuration.md):**

1. **`server.max_concurrent` missing** — config.go line 29, example.yaml lines 28-30. Not in configuration.md Server Configuration table.

2. **`server.cors` section missing entirely** — `CORSConfig` struct (config.go lines 34-39), example.yaml lines 33-43. Not documented at all.

3. **`upstream.insecure_skip_verify` missing** — config.go line 161. Only `url` and `timeout` appear in the Upstream table (configuration.md lines 228-234).

4. **`user_management.enabled` field does not exist in the struct** — `UserMgmtConfig` (config.go lines 142-145) has only `password_length` and `credential_ttl`. The `enabled` field shown in configuration.md line 194 is stale/wrong. Check: `config/config.example.yaml` lines 347-362 confirms `user_management` section has no `enabled:` field (the `enabled:` in the example file is a blank/orphaned line from a struct change).

5. **LDAP section entirely missing** — configuration.md has no LDAP section. `LDAPConfig` struct (config.go lines 84-119) has 15+ fields.

6. **`session.cookie_path` missing** — config.go line 127. Not in Session Configuration table (configuration.md lines 95-108).

7. **`observability` section missing** — `ObservabilityConfig` struct (config.go lines 165-184). Not documented in configuration.md.

8. **`cache.redis_password` and `cache.redis_db` missing** — config.go lines 133-134. Only `backend`, `redis_url`, `credential_ttl`, `encryption_key` appear in the cache table.

9. **`oidc.user_identity_claim` not in configuration.md** — `OIDCConfig` has this field (config.go line 50), but configuration.md has no OIDC section at all (only in authentication/oidc-authentication.md).

**Correct table rows to add/fix:**

Server table (add):
```
| `max_concurrent` | 0 (unlimited) | Max concurrent requests; 503 when exceeded |
| `cors.allowed_origins` | [] | List of allowed CORS origin domains |
```

Session table (add):
```
| `cookie_path` | / | Cookie path |
```

Cache table (add):
```
| `redis_password` | - | Redis authentication password |
| `redis_db` | 0 | Redis database number (0-15) |
```

New LDAP section (condensed from config.go lines 84-119):
```yaml
ldap:
  enabled: true
  url: ${LDAP_URL}            # ldap:// or ldaps://
  bind_dn: ${LDAP_BIND_DN}
  bind_password: ${LDAP_BIND_PASSWORD}  # Must be ${ENV_VAR} reference
  connection_timeout: 10s
  tls_mode: ldaps             # none | ldaps | starttls
  tls_skip_verify: false
  search_base: ${LDAP_SEARCH_BASE}
  search_filter: "(sAMAccountName={username})"
  group_search_base: ${LDAP_GROUP_SEARCH_BASE}
  group_search_filter: "(member={user_dn})"
  username_attribute: sAMAccountName
  email_attribute: mail
  display_name_attribute: displayName
  group_name_attribute: cn
  required_groups: []         # Optional access gate
```

**user_management.enabled fix:** Remove `enabled` from the table and config snippet. The section should show only `password_length` and `credential_ttl`. Note that user management is always on; it is activated by providing the `elasticsearch.admin_user` and `admin_password`.

---

### `docs/docs/authentication/overview.md` (stale — LDAP missing from overview)

**Current state:** Lists only OIDC and Basic Auth (table line 13-15). LDAP is implemented in `internal/auth/engine.go` lines 26-28 (`ldapProvider *LDAPProvider`, `ldapEnabled bool`) but not shown in this overview.

**Pattern to follow:** The existing method table (overview.md lines 12-15):
```markdown
| Method | Use Case | Session | Best For |
|--------|----------|---------|----------|
| **OIDC** | Interactive browser authentication | Yes (cookie-based) | Human users, SSO |
| **Basic Auth** | Programmatic/API access | No (stateless) | CI/CD, monitoring, scripts |
```

**Add row:**
```markdown
| **LDAP / AD** | Corporate directory authentication | No (stateless) | Existing AD environments |
```

**Add to "Authentication Endpoints" section:** LDAP has no special endpoint — it reuses the Basic Auth header path. A note clarifying this (LDAP uses `Authorization: Basic` header just like local users, auth method is selected automatically) should be added.

**Auth flow priority order** (from `internal/auth/engine.go` — read lines 1-80):
Session → Basic Auth check → LDAP (when no local user match) → OIDC redirect.
The overview mermaid diagram doesn't show LDAP. A branch from `CheckBasic` to "Try local_users → if not found, try LDAP" should be added or noted.

---

### `docs/docs/authentication/ldap-authentication.md` (NEW FILE)

**This file does not exist.** Must be created.

**Analog to copy structure from:** `docs/docs/authentication/local-users-basic-auth.md` (7.6K, full read above).

**Frontmatter pattern** (copy from local-users-basic-auth.md lines 1-4):
```markdown
---
sidebar_label: LDAP Authentication
sidebar_position: 4
---
```

**Required sections** (derive content from `internal/auth/ldap.go` and `config/config.example.yaml` lines 143-210):

1. **Overview** — LDAP/AD auth via Basic Auth header; coexists with local_users (local checked first); uses service account bind + user bind.
2. **Configuration** — Full LDAP config block. Source: `config/config.example.yaml` lines 143-210 and `internal/config/config.go` lines 84-119.
3. **TLS Modes** — Three modes with guidance:

| TLS Mode | Value | Description |
|---|---|---|
| LDAP over TLS | `ldaps` | Port 636; recommended for production |
| STARTTLS upgrade | `starttls` | Upgrades plain connection to TLS |
| Plaintext (dev only) | `none` | No encryption; never use in production |

4. **Attribute Mapping** — Two ways to configure: per-field (`username_attribute`, `email_attribute`, etc.) or `attribute_mapping` map. Source: config.go lines 106-115.
5. **Group Search** — Optional; omit `group_search_base` + `group_search_filter` to skip. Source: config.go lines 102-103, example.yaml lines 179-183.
6. **Required Groups** — Access gate. Source: example.yaml lines 204-207.
7. **Active Directory vs. OpenLDAP examples** — Two complete config examples.
8. **Username Normalisation** — Keyline trims, lower-cases, and replaces unsupported characters. Source: `internal/auth/ldap.go` lines 225-232.
9. **bind_password enforcement** — Must be `${ENV_VAR}` reference (not plaintext). Source: ldap.go lines 115-128.
10. **Troubleshooting** — Common errors: connection failure, bind failure, user not found, TLS errors.

**Key config excerpt to include** (from example.yaml lines 538-560):
```yaml
# Active Directory example
ldap:
  enabled: true
  url: ldaps://ad.corp.yourdomain.com:636
  bind_dn: CN=keyline-svc,OU=ServiceAccounts,DC=corp,DC=yourdomain,DC=com
  bind_password: ${LDAP_BIND_PASSWORD}
  tls_mode: ldaps
  search_base: DC=corp,DC=yourdomain,DC=com
  search_filter: "(sAMAccountName={username})"
  group_search_base: OU=Groups,DC=corp,DC=yourdomain,DC=com
  group_search_filter: "(member={user_dn})"
  required_groups:
    - keyline-users

# OpenLDAP example
ldap:
  enabled: true
  url: ldap://ldap.yourdomain.com:389
  bind_dn: cn=keyline-svc,dc=yourdomain,dc=com
  bind_password: ${LDAP_BIND_PASSWORD}
  tls_mode: starttls
  search_base: dc=yourdomain,dc=com
  search_filter: "(uid={username})"
  attribute_mapping:
    username: uid
    email: mail
    displayName: cn
```

---

### `docs/docs/user-management/dynamic-user-management.md` (accuracy review)

**Source of truth:** `internal/usermgmt/manager.go` + `internal/config/config.go` lines 142-145.

**Issue found:** `user_management.enabled` appears in the doc config block (line 103) but the `UserMgmtConfig` struct has no `Enabled` field (config.go lines 142-145). The config example also shows it as blank. This is a stale field reference — remove `enabled: true` from the config snippet and replace with a note that user management activates when `elasticsearch.admin_user` is configured.

**Otherwise accurate:** The mermaid diagrams, component descriptions, cache key format `keyline:user:{username}:password`, AES-256-GCM encryption, and metric names all match `internal/usermgmt/` implementations.

**No structural changes needed** beyond the `enabled` field fix and verifying metric names match `internal/usermgmt/metrics.go`.

---

### `docs/docs/user-management/role-mappings.md` (accuracy review)

**Source of truth:** `internal/auth/` + `config/config.example.yaml` lines 263-319.

**Accurate:** Pattern matching logic (evaluate ALL mappings, collect ALL matches, deduplicate) matches `internal/usermgmt/rolemapper.go`. Wildcard support, `default_es_roles` fallback, and deny-if-no-match behavior are all correct.

**Minor issue:** The `claim` field documentation says values are `groups` or `email` but the config struct (`RoleMapping`, config.go lines 62-66) has `Claim string` with no enum constraint. The actual OIDC implementation may support any claim name. Document as "typically `groups` or `email`" rather than "only `groups` or `email`".

**No new content needed** — this doc is well-matched to the implementation.

---

### `docs/docs/deployment/docker.md` (accuracy review)

**Source of truth:** The actual image reference in the file itself + README.md.

**Issues found:**

1. **Elasticsearch image version** (line 162): `docker.elastic.co/elasticsearch/elasticsearch:9.3.1` — this is a very specific pin. Verify this version is intentional or use `8.x` as a more stable example. (Planner note: leave as-is unless there's a reason to change — the version appears to be real.)

2. **`upstream.insecure_skip_verify` env-var** not shown in docker.md Environment Variables table (lines 230-237). The full stack compose example would need it for self-signed ES certs in dev.

3. **Image reference is correct:** `ghcr.io/wasilak/keyline:latest` matches README.md line 44.

4. **Health check endpoint discrepancy:** docker.md uses `/healthz` (line 49, 76); quick-start.md uses `/_health` (line 117). Need to verify which is correct from the actual server implementation.

**Pattern for new env-var table row** (copy from existing table, lines 230-237):
```markdown
| `LDAP_BIND_PASSWORD` | No | LDAP bind password (if using LDAP auth) |
```

---

### `docs/docs/getting-started/quick-start.md` (accuracy review)

**Issues found:**

1. **Health endpoint** (line 117): `curl http://localhost:9000/_health` with expected response `"version": "1.0.0"`. Needs verification — if v2.0 returns a different version or path, fix accordingly.

2. **`user_management.enabled: true`** (line 97) — same stale field issue as in dynamic-user-management.md. The struct has no `enabled` field.

3. **`upstream.insecure_skip_verify: true`** (line 105) is present in the config snippet. This is fine for a quick-start guide (dev scenario with self-signed certs).

4. **`elasticsearch.insecure_skip_verify: true`** (line 95) — correctly shown for dev/quick-start.

**Otherwise accurate.** The quick-start config structure matches the canonical config structs.

---

### `docs/docusaurus.config.js` (check for placeholder URLs/org refs)

**Current state:** All org/URL references are correct.
- `organizationName: 'wasilak'` (line 17) — correct.
- `projectName: 'keyline'` (line 18) — correct.
- `editUrl: 'https://github.com/wasilak/keyline/tree/main/docs/'` (line 52) — correct.
- GitHub link in navbar: `https://github.com/wasilak/keyline` (line 94) — correct.
- Issues link in footer: `https://github.com/wasilak/keyline/issues` (line 129) — correct.
- `url: 'https://wasilak.github.io'` (line 11) — correct.

**Issue found — stale version label** (line 56):
```javascript
label: '1.0.x (Latest)',
```
This should be `2.0.x (Latest)` since the release is v2.0. This is a placeholder that wasn't updated.

**Issue found — duplicate `markdown:` key** (lines 23-24 and 36-37):
```javascript
markdown: {
  hooks: { onBrokenMarkdownLinks: 'warn' },
},
// ... later ...
markdown: {
  mermaid: true,
  format: 'mdx',
},
```
JavaScript object literal with duplicate keys — the second definition silently overwrites the first. `onBrokenMarkdownLinks` is being dropped. Merge into a single `markdown:` block:
```javascript
markdown: {
  mermaid: true,
  format: 'mdx',
  hooks: {
    onBrokenMarkdownLinks: 'warn',
  },
},
```

---

## Shared Patterns

### Config field naming convention
**Source:** `internal/config/config.go` (all `mapstructure` tags)
**Apply to:** All config snippets in all docs files

The canonical field names (snake_case) from the struct tags are:
- `user_management` (not `userManagement`)
- `password_length`, `credential_ttl`
- `tls_skip_verify`, `tls_mode`
- `insecure_skip_verify` (ES and upstream)
- `admin_user`, `admin_password`
- `otel_tls_skip_verify`

### URL placeholder pattern
**Source:** Decision D-06 from CONTEXT.md
**Apply to:** All new config examples in all docs files
```
# Use:    https://auth.yourdomain.com/auth/callback
# Avoid:  https://auth.example.com/auth/callback
```
Exception: Existing YAML *content* examples in RELEASE-NOTES.md (D-08 — keep `example.com` in YAML snippets).

### Frontmatter pattern for auth docs
**Source:** `docs/docs/authentication/local-users-basic-auth.md` lines 1-4
```markdown
---
sidebar_label: [Human-readable label]
sidebar_position: [integer]
---
```

### `your-org` → `wasilak` replacement
**Source:** Decision D-07 from CONTEXT.md + `go.mod` module identity
**Apply to:** RELEASE-NOTES.md only (no other files contain `your-org`)
**Pattern:** Simple string replacement, `your-org` → `wasilak`

---

## No Analog Found

Files where content must be written from scratch using source code reading:

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `docs/docs/authentication/ldap-authentication.md` | doc, guide (NEW) | reference | File does not exist; LDAP was a v2.0 addition with no existing doc |

---

## Critical Accuracy Findings (summary for planner)

These are correctness bugs — not style issues — that must be fixed:

| File | Issue | Source of Truth |
|---|---|---|
| `docs/docs/configuration.md` | `user_management.enabled` field doesn't exist in struct | `config/config.go:142-145` |
| `docs/docs/configuration.md` | CORS section entirely absent | `config/config.go:34-39` |
| `docs/docs/configuration.md` | LDAP section entirely absent | `config/config.go:84-119` |
| `docs/docs/configuration.md` | Observability section entirely absent | `config/config.go:165-184` |
| `docs/docs/configuration.md` | `cache.redis_password`, `cache.redis_db` missing from table | `config/config.go:133-134` |
| `docs/docs/user-management/dynamic-user-management.md` | `user_management.enabled: true` in snippet is stale | `config/config.go:142-145` |
| `docs/docs/getting-started/quick-start.md` | `user_management.enabled: true` in snippet is stale | `config/config.go:142-145` |
| `docs/docusaurus.config.js` | Duplicate `markdown:` key silently drops `onBrokenMarkdownLinks` | JS object semantics |
| `docs/docusaurus.config.js` | Version label shows `1.0.x (Latest)` instead of `2.0.x (Latest)` | Release state |
| `RELEASE-NOTES.md` | LDAP auth not mentioned as a v2.0 new feature | `internal/auth/ldap.go` |
| `RELEASE-NOTES.md` | Docker image `keyline:v2.0.0` should be `ghcr.io/wasilak/keyline:v2.0.0` | `README.md:44`, `docs/deployment/docker.md:19` |
| `README.md` | Architecture diagram labels OIDC+Basic only; LDAP missing | `internal/auth/engine.go:26-28` |

---

## Metadata

**Analog search scope:** `docs/docs/`, `internal/auth/`, `internal/config/`, `internal/usermgmt/`, `config/`
**Files scanned:** 18
**Pattern extraction date:** 2026-05-17
