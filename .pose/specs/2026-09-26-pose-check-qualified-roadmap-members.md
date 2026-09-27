---
slug: pose-check-qualified-roadmap-members
status: done
created_at: 2026-09-26
supersedes:
depends_on: pose-federated-external-milestone-members
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:check-qualified-roadmap-members
completed_at: 2026-09-27
---

# Spec: `pose check` validates qualified roadmap members through the graph

## 1. Intent

### Goal
Make `pose check` validate `xref:` milestone members and `after` entries of a
roadmap through the authorized project graph, as it already does for a spec's
`depends_on`.

### Business value
After Harne8 retired its ABM coordinators, its roadmaps name the executors as
`xref:proj.pose-dist/spec:*`. `pose check --strict` then failed with 11 errors:
`missing spec: xref:...` for every member and `after references a missing
milestone: xref:...` for every qualified `after` entry, because the roadmap
check only looked for local specs and milestones.

### Constraints
An unresolvable, unauthorized or cyclic qualified entry must still fail.
Ownership of external specs stays with federated acceptance.

### Non-goals
Do not change local member or `after` validation.

## 2. Requirements

- R1: A resolvable `xref:` milestone member or `after` entry passes `pose check`.
- R2: An unresolvable one fails with the graph's reason.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/cli/check.go` (`checkRoadmaps`, `checkFederatedRoadmapRef`).

### Artifacts
- created: .pose/specs/2026-09-26-pose-check-qualified-roadmap-members.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/spec_transfer_test.go

### Delivery targets
- contract:check-qualified-roadmap-members module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Route qualified roadmap entries through `ArtifactResolver.ValidateGraph`.
Rollback is a revert.

## 4. Tasks

- [x] Reproduce the 11 errors on Harne8 after the reconciliation.
- [x] Write the regression and prove it fails without the change.
- [x] Validate qualified entries through the graph.
- [x] Run the matrix, review and close.

## 5. Decisions

Mirrors the existing `depends_on` branch of `checkSpecs`.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1/R2 | matrix check `spec-authority-transfer-integration` | Resolvable member passes, unknown project reported. |
| Harne8 consumer | `pose check --strict` in Harne8 | Passes with the candidate; fails with 11 errors on the previous pin. |

### Execution log
2026-09-26: on Harne8 after the reconciliation, `pose check --strict` with the
pinned `cf0184b` binary failed with 11 roadmap errors and passed with a
candidate carrying this change. The regression failed with the qualified
branches disabled. `go test ./...` and `go vet ./...` pass.

At `ecd0302` the full matrix passed 29/29. Bundle `rvb-f000822add044cfb` was
approved by attestation `rva-0aafac0512647390`, recorded by the agent under
explicit authorization from the user to self-attest; superseded reviews were
resealed and reattested.

### Requirement trace
- R1 [satisfied] test:TestSpecTransferCheckValidatesQualifiedRoadmapMembers
- R2 [satisfied] test:TestSpecTransferCheckValidatesQualifiedRoadmapMembers

## 7. Final Report

### Delivered scope
`pose check` accepts resolvable qualified roadmap members and `after` entries
and reports unresolvable ones with the graph's reason.

### Residual risks
None identified.

### Follow-ups
