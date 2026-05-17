# Phase 2: Documentation Update - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-17
**Phase:** 2-documentation-update
**Areas discussed:** Docusaurus docs scope, README update depth

---

## Docusaurus Docs Scope

### How thorough should the docs/ review be?

| Option | Description | Selected |
|--------|-------------|----------|
| Placeholder pass only | Just find and fix any your-org/yourusername/example placeholder strings | |
| Accuracy review | Check config field names, feature descriptions, and examples match actual v2.0 code | ✓ |
| Skip docs/ entirely | Phase 2 covers only README.md and RELEASE-NOTES.md | |

**User's choice:** Accuracy review

---

### Which docs/ sections are most likely to be stale?

| Option | Description | Selected |
|--------|-------------|----------|
| Configuration reference | docs/docs/configuration.md — config field names and defaults | ✓ |
| Authentication section | docs/docs/authentication/ — OIDC, Basic Auth, LDAP guides | ✓ |
| User management section | docs/docs/user-management/ — headline v2.0 feature | ✓ |
| Deployment / getting started | docs/docs/deployment/ and getting-started/ | ✓ |

**User's choice:** All four sections

---

### When accuracy review finds a gap (e.g., LDAP undocumented)?

| Option | Description | Selected |
|--------|-------------|----------|
| Write the missing content | Write short practical explanation + config example. Treat missing docs as defect. | ✓ |
| Fix and flag only | Fix stale content; flag missing sections in SUMMARY.md | |
| You decide | Use judgment on whether to write or flag | |

**User's choice:** Write the missing content

---

### Should docusaurus.config.js also be checked?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — check config too | Include docusaurus.config.js in the placeholder/org scan | ✓ |
| No — content docs only | Focus only on Markdown content files | |

**User's choice:** Yes — check config too

---

## README Update Depth

### What's the primary gap to fix in README?

| Option | Description | Selected |
|--------|-------------|----------|
| Add 'What's New in v2.0' section | Dedicated new section listing v2.0 additions | |
| Expand existing feature sections | Find thin/missing coverage and expand inline. No new top-level section. | ✓ |
| Both — new section + fill gaps | New highlights section AND patch thin descriptions | |

**User's choice:** Expand existing feature sections

---

### The README has `example.com` in a config URL — acceptable?

| Option | Description | Selected |
|--------|-------------|----------|
| Keep example.com — it's a standard placeholder | IANA-reserved, fine for docs | |
| Use realistic URL pattern | Replace with https://auth.yourdomain.com/auth/callback style | ✓ |

**User's choice:** Use realistic URL pattern (e.g., `https://auth.yourdomain.com/auth/callback`)

---

## Claude's Discretion

- Release date for RELEASE-NOTES v2.0.0 header: not discussed. Agent may leave "TBD" since the actual release date should be set by the maintainer.
- Depth of new content for undocumented v2.0 features: write practical, accurate content derived from reading actual code and config structs.

## Deferred Ideas

None — discussion stayed within phase scope.
