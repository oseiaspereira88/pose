---
slug: pose-dist-adopts-spec-authority-transfer
status: done
created_at: 2026-09-26
completed_at: 2026-09-26
supersedes:
depends_on: pose-spec-authority-transfer, pose-federated-spec-dependency-acceptance
priority: 0
components:
task_type: refactor
delivers:
---

# Spec: this repository adopts the spec authority transfer contract

## 1. Intent

### Goal
Adopt review policy schema 4 in this instance, so it can be the destination of
the governed spec transfers Harne8 needs to reconcile its ABM specs.

### Business value
A transfer requires schema 4 in both source and destination
(`requireSpecTransferCapability`). Harne8 is ready to adopt it, but without the
same adoption here every preview or apply is refused with
`spec-authority-transfer-capability-not-adopted`, which blocks the
reconciliation that precedes the 6.0.0 contracts.

### Constraints
Instance policy only: the shipped scaffold keeps its current schema, since
adopting a contract is a decision of each instance. Keep sealed bundles and
attestations unchanged; renew the reviews this change supersedes with fresh
evidence. knowledge:multirepo-planning records the consumer sequence.

### Non-goals
Do not perform any transfer, change the scaffold or engine code, or alter
historic review records.

## 2. Requirements

### Functional
- R1: `.pose/policy/review.json` declares schema 4 with
  `qualified_artifact_refs_version: 1` and `spec_authority_transfer_version: 1`,
  and every other key keeps its value.
- R2: With both instances adopted, the engine resolves qualified authority,
  refuses a `new-spec --task xref:` that would create a shadow spec, and
  reports no new structural error.
- R3: The multirepo foundation specs, milestones and roadmap regain fresh
  approved reviews after the change, so consumers can federate them again.

### Security and compatibility
Engines older than the transfer contract refuse schema 4 instead of operating
on transfer state. Rollback is reverting the policy commit before any transfer
is applied; after an applied transfer, do not restore retired specs.

## 3. Technical Plan

### Affected areas
Instance governance only.

### Artifacts
- created: .pose/specs/2026-09-26-pose-dist-adopts-spec-authority-transfer.md
- modified: .pose/policy/review.json

### Approach and rollback
Rehearsed first in detached worktrees of both repositories: schema 4 in both,
gitlink and trust pin renewed in the Harne8 copy. Apply the same change here,
regenerate evidence, renew the superseded foundation reviews and let Harne8 pin
the resulting revision. Revert the policy commit to roll back while no transfer
exists.

## 4. Tasks

- [x] Rehearse the adoption in disposable copies of both repositories.
- [x] Adopt schema 4 in this instance.
- [x] Renew the superseded foundation reviews with fresh evidence.
- [x] Review and close.

## 5. Decisions

Decision D3 of the Harne8 release plan activates the transfer policy at
adoption closeout, after spec-scope federation (`pose-federated-spec-dependency-acceptance`)
is enforced. The contract is the one accepted in the
[identity ADR](../adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md).

## 6. Validation

### Strategy
Risk: medium; governance contract of this instance. The rehearsal measures the
blast radius before the real change.

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Structural gate, R1/R2 | `pose check --strict` | No error beyond the pre-existing missing knowledge artifact. |
| Shadow-spec negative, R2 | `pose new-spec probe --task xref:proj.pose-dist/spec:pose-agent-project-context` in Harne8 | Refused; no file written. |
| Foundation reviews, R3 | `pose review verify` for the nine foundation scopes | Fresh and approved. |
| Roadmap, R3 | `pose roadmap-check pose-multirepo-foundation --strict` | Terminal. |

### Execution log
2026-09-26: rehearsal in detached worktrees (pose-dist `3941bd8`, Harne8
`54323f52`), schema 4 committed in both and the Harne8 gitlink and trust pin
renewed. Closeout of the closed scopes stayed terminal and fresh in both
repositories. `review verify`, the path federated source proof takes, reported
the foundation scopes superseded, so Harne8 saw nine
`source-review-not-approved` until they are renewed. `pose check --strict`
added no error: pose-dist kept its pre-existing one, and Harne8's three were
links into the worktree's empty submodule. `new-spec --task xref:` was refused
with no file written, and `pose context` resolved `proj.pose-dist` authority.
At `e72f2d6` the full matrix passed 29/29, the regenerated index differed from
the committed one only by this spec's change set, and `pose check --strict`
kept only the pre-existing missing knowledge artifact. The four foundation
specs, four milestones, roadmap and the two federation specs were resealed and
reattested fresh and approved; `roadmap-check pose-multirepo-foundation
--strict` is terminal.

### Requirement trace
- R1 [satisfied] report:.pose/specs/2026-09-26-pose-dist-adopts-spec-authority-transfer.md
- R2 [satisfied] test:TestSpecTransferNegative
- R3 [satisfied] integration:pose-mcp/go/federated-roadmap-acceptance-integration

## 7. Final Report

### Delivered scope
This instance declares review policy schema 4 with qualified references and
spec authority transfer, and its foundation reviews are current under it.

### Residual risks
No transfer has been applied. Once one is, rollback no longer means reverting
the policy: retired specs must not be restored.

### Follow-ups
