# Phase 2: Documentation Update - Context

**Gathered:** 2026-05-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Update README.md and RELEASE-NOTES.md to accurately reflect v2.0 features and correct all placeholder org/URL references. Conduct an accuracy review of the Docusaurus docs site (docs/) — verifying config field names, feature descriptions, and examples against actual v2.0 code, and writing missing content for undocumented v2.0 features.

</domain>

<decisions>
## Implementation Decisions

### Docusaurus docs scope
- **D-01:** Accuracy review — not just a placeholder pass. Check that config field names, feature descriptions, and examples in docs/ match the actual v2.0 code (config structs, README references). Fix anything stale or wrong.
- **D-02:** All four sections get focused attention: configuration reference, authentication section (LDAP was added in v2.0), user management section (headline v2.0 feature), and deployment/getting-started.
- **D-03:** When a v2.0 feature is completely absent from docs, write the missing content (short practical explanation + config example). Treat missing docs as a defect, not a flag-for-later.
- **D-04:** Also check docusaurus.config.js for placeholder URLs or org references (GitHub URLs, site metadata).

### README update depth
- **D-05:** Expand existing feature sections (not a new top-level "What's New in v2.0" section). Find thin or missing coverage of v2.0 features within the current structure and expand inline.
- **D-06:** Replace `example.com` URLs in config examples with a more realistic but still generic pattern — e.g., `https://auth.yourdomain.com/auth/callback` rather than `https://auth.example.com/auth/callback`.

### RELEASE-NOTES placeholders
- **D-07:** Replace all `your-org` references with `wasilak` in RELEASE-NOTES.md: GitHub URLs (`https://github.com/your-org/keyline` → `https://github.com/wasilak/keyline`), email (`security@your-org.com` → appropriate wasilak contact), and any other org references.
- **D-08:** `example.com` in YAML config snippet examples within RELEASE-NOTES (e.g., `testuser@example.com`, `*@admin.example.com`) — these are intentional placeholder values in config examples, acceptable to keep.

### Claude's Discretion
- Release date for RELEASE-NOTES v2.0.0 header: not discussed — agent may use "2026-05-17" or "TBD" (leave TBD if a specific release date should be set by the maintainer, not the docs update).
- Depth of new content for undocumented v2.0 features in docs/ — write practical, accurate content derived from reading the actual code and config structs. Quality over quantity.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Source files to verify accuracy against
- `go.mod` — module identity (now `github.com/wasilak/keyline`)
- `config/config.example.yaml` — canonical config field names and structure
- `config/user-management-example.yaml` — user management config reference
- `internal/auth/` — auth engine, OIDC, Basic Auth, LDAP implementations
- `internal/usermgmt/` — dynamic user management implementation
- `internal/session/` — session backend implementations (Redis, in-memory)

### Documentation files to update
- `README.md` — 140 lines, has architecture diagram; expand v2.0 feature sections
- `RELEASE-NOTES.md` — 406 lines; fix `your-org` placeholder refs, leave `example.com` in YAML snippets
- `docs/docs/configuration.md` — config reference; verify against actual config structs
- `docs/docs/authentication/` — OIDC, Basic Auth, LDAP guides
- `docs/docs/user-management/` — dynamic user management docs (headline v2.0 feature)
- `docs/docs/deployment/` — Docker, compose examples; check env-var references
- `docs/docs/getting-started/` — onboarding content; check for v2.0 accuracy
- `docs/docusaurus.config.js` — check for placeholder GitHub URLs or org metadata

### Requirements
- `REQUIREMENTS.md` — DOC-01, DOC-02 (this phase's requirements)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `config/config.example.yaml` (17.3K) — comprehensive config reference; use as ground truth for all config field names in docs
- `config/user-management-example.yaml` (14.8K) — user management config; use for user-management docs accuracy check

### Established Patterns
- Docusaurus docs are in `docs/docs/` with topic-based directory structure (authentication/, deployment/, user-management/, etc.)
- README follows: headline → architecture diagram → what changed from elastauth → feature sections — expand within this structure, don't restructure

### Integration Points
- GitHub module identity is now `github.com/wasilak/keyline` (confirmed by Phase 1); all docs references should use this
- Docusaurus publishes to `wasilak.github.io/keyline` (per README); any self-referential links in docs should point there

</code_context>

<specifics>
## Specific Ideas

- Config example URLs: use `https://auth.yourdomain.com/auth/callback` pattern (more instructive than bare `example.com`)
- README feature expansion: v2.0 features to cover — dynamic Elasticsearch user management, LDAP auth with TLS modes (ldaps/starttls/plaintext), role mapping (pattern-based, groups→roles), AES-256-GCM encrypted credential caching, Redis session backend, circuit breaker on ES client, CORS allowed origins, env-var enforcement for sensitive fields

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 2-documentation-update*
*Context gathered: 2026-05-17*
