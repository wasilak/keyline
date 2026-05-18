# Retrospective: v0.2.0 — Observability & Integration

**Date:** 2026-05-18
**Duration:** 1 day (2026-05-17 → 2026-05-18)
**Phases:** 5 (Phases 3–7)
**Commits:** 7 · 34 files · +2573 / -96 lines

---

## What Went Well

**Scope was tight and deliverable-oriented.** Every requirement mapped cleanly to a phase, and every phase had a concrete artifact to ship (code, doc, or design). Nothing ambiguous made it into the milestone.

**Circular import hazard caught early.** Moving `ESAPICallsTotal` to `internal/elasticsearch` before it became a runtime surprise was the right call. The lesson: metrics should live where their instrumented code lives, not in a central observability package.

**Audit log design is clean.** Guarding trace correlation with `span.IsRecording()` means the audit schema is identical whether OTel is enabled or not — just no trace IDs when it isn't. Avoids a conditional schema that would be painful to query across environments.

**Secan spike was the right scope.** "Document and design" rather than "implement" saved a sprint on something architecturally uncertain. Option C (hybrid) emerged naturally from the constraints analysis — wouldn't have been obvious without working through all three topologies.

**Phase retroactive reconciliation worked.** The milestone was executed without GSD tooling per phase, but the retroactive reconciliation produced clean phase dirs and an accurate STATE.md. Good fallback pattern for "just ship it" sessions.

---

## What Was Hard

**Two-instance Secan limitation is real.** Option C requires two Keyline processes: one for browser→Secan forwardAuth, one as Secan→ES proxy. That's operational overhead. It surfaced late in the spike — earlier constraint enumeration would have caught it sooner.

**Metrics placement convention implicit until it broke.** The circular import forced an explicit decision on where subsystem metrics live. That convention should be documented upfront for any new subsystem added in v2.2+.

---

## What to Carry Forward

| Item | Action |
|------|--------|
| Metrics convention: subsystem metrics live in their package | Add to contributing notes or CLAUDE.md before v2.2 |
| Secan two-instance limitation | Document operational runbook before implementing Option C |
| OTel log bridge is opt-in and isolated | Safe to keep as default-off; revisit only if loggergo upstream changes |
| `clientWithCircuitBreaker.DeleteUser` stub bug | Pre-existing, non-blocking — flag in a future issue before v2.2 user management expansion |

---

## Backlog Candidates for v2.2

From the PROJECT.md backlog (no priority set — discuss at next milestone kickoff):

- DEPL-01: Helm chart
- DEPL-02: Kubernetes operator
- OBS-01: Grafana dashboard
- OBS-02: Alerting runbook
- SECAN-IMPL: Implement Option C (requires multi-listener support design first)
