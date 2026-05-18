# Roadmap: Keyline

## Shipped

| Milestone | Name | Phases | Status | Shipped |
|-----------|------|--------|--------|---------|
| v1.0 | Foundation | — | Complete | ~2026-05 |
| v2.0 | Ship | 1–2 | Complete | 2026-05-17 |
| v0.2.0 | Observability & Integration | 3–7 | Complete | 2026-05-18 |

Archived roadmaps and requirements: `.planning/milestones/`

---

## Backlog (candidates for v2.2+)

### Deployment

- **DEPL-01**: Helm chart published to a Helm repository for `helm repo add` + `helm install`
- **DEPL-02**: Kubernetes operator for declarative Keyline configuration

### Observability

- **OBS-01**: Grafana dashboard published alongside Keyline docs
- **OBS-02**: Alerting runbook for common Keyline failure modes

### Integration

- **SECAN-IMPL**: Implement Secan Option C integration (design validated in v0.2.0; requires multi-listener support or two-instance deployment)

---

## Future (v3+)

- Multi-cluster routing — single ES cluster target per instance today; architectural change
- Admin UI — browser-based management dashboard; high complexity

---
*Last updated: 2026-05-18 — v0.2.0 milestone closed*
