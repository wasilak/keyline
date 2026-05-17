# Requirements: Keyline

**Defined:** 2026-05-17
**Core Value:** Authenticated users get their own Elasticsearch identities automatically — real accountability and auditing without per-user pre-configuration.

## v2.0 Requirements

Requirements for the v2.0 release milestone. Each maps to a roadmap phase.

### Module Identity

- [ ] **MOD-01**: Developer can build keyline with the correct Go module name (`github.com/wasilak/keyline`) — all import paths updated consistently
- [ ] **MOD-02**: `go.mod` specifies a valid Go version that matches the actual minimum required by the codebase

### Documentation

- [x] **DOC-01**: README accurately describes v2.0 features (dynamic user management, LDAP, role mapping, Redis caching)
- [x] **DOC-02**: RELEASE-NOTES.md contains no placeholder URLs or org references (`your-org` → `wasilak`); all links point to correct GitHub locations

## Future Requirements

Features deferred beyond v2.0.

### Deployment

- **DEPL-01**: Helm chart published to a Helm repository for `helm repo add` + `helm install`
- **DEPL-02**: Kubernetes operator for declarative Keyline configuration

### Observability

- **OBS-01**: Grafana dashboard published alongside Keyline docs
- **OBS-02**: Alerting runbook for common Keyline failure modes

## Out of Scope

| Feature | Reason |
|---------|--------|
| Admin UI | Browser-based management dashboard; high complexity, not core to auth proxy value |
| Multi-cluster routing | Single ES cluster target per instance; architectural change deferred to v3+ |
| Re-bind failure recovery (LDAP) | Current behavior is correct per LDAP spec; documentation-only if needed |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MOD-01 | Phase 1 | Pending |
| MOD-02 | Phase 1 | Pending |
| DOC-01 | Phase 2 | Complete |
| DOC-02 | Phase 2 | Complete |

**Coverage:**
- v2.0 requirements: 4 total
- Mapped to phases: 4
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-17*
*Last updated: 2026-05-17 after initial definition*
