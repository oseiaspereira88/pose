---
slug: pose-federated-external-milestone-members
status: in-progress
created_at: 2026-09-26
supersedes:
depends_on: pose-roadmap-gate-scopes-milestones-and-external-members
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:federated-external-milestone-members
---

# Spec: Milestones whose members belong to another project

## 1. Intent

### Goal
Let milestone closeout, review and readiness accept members qualified with
another project, judging them through federated acceptance only.

### Business value
In a rehearsal of Harne8's ABM reconciliation, `closeout-check
milestone:pose-abm-foundation/integrity` failed with `invalid spec slug
"xref:proj.pose-dist/spec:pose-abm-review-soundness"`: closeout, lifecycle,
scope digest, review plan, bundle children and readiness all read each member
as a local slug. Once external, the milestone bundle also refused with no
local change set, although the implementation is sealed upstream.

### Constraints
Fail closed: without a federated resolver an external member blocks. Local
members keep every existing check. Do not change validation-matrix.json.

### Non-goals
Do not change federated traversal or roadmap cut criteria.

## 2. Requirements

- R1: Closeout of a milestone with an external member does not fail; with a resolver it takes no local child for that member and relies on federated acceptance, and without one it blocks with `external member <ref> needs federated acceptance`.
- R2: Review plan, scope digest and bundle children skip external members; a milestone whose members are all external seals with no local subject when a resolver is configured.
- R3: Readiness of a milestone treats an external member as satisfied only when the federated resolver resolves it done.
- R4: Revoked trust still blocks the milestone of an external member.

## 3. Technical Plan

### Affected areas
`review_closeout.go` (`milestoneMember`, `externalMemberDone`), `review_bundle.go`,
`review_plan.go`, `readiness.go`.

### Artifacts
- created: .pose/specs/2026-09-26-pose-federated-external-milestone-members.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/readiness.go
- modified: pose-mcp/internal/pose/federated_acceptance_test.go

### Delivery targets
- contract:federated-external-milestone-members module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
One helper classifies a member as local (unqualified or own project) or
external; every milestone reader uses it. Rollback is a revert.

## 4. Tasks

- [x] Reproduce the refusals in the reconciliation rehearsal.
- [x] Write the regression and see it fail on each refusal in turn.
- [x] Route every milestone reader through the helper.
- [ ] Run the matrix, review and close.

## 5. Decisions

Complements [roadmap gate scopes](2026-09-26-pose-roadmap-gate-scopes-milestones-and-external-members.md),
which fixed the CLI gate; this fixes the Store readers under it.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1-R4 | matrix check `federated-roadmap-acceptance-integration` | Closeout, bundle and readiness with and without resolver, and revocation. |
| Module matrix | `pose validate --strict --module pose-mcp --report` | Required checks pass. |

### Execution log
2026-09-26: the regression failed first with `invalid spec slug` from closeout,
then from the review plan, then with `no immutable attributed change set`
from the bundle, each removed by routing that reader through the helper. With
the change the rehearsal's four implementation milestones report ready
federated acceptance and prepare bundles without blockers; `go test ./...`
and `go vet ./...` pass.

### Requirement trace
- R1 [satisfied] test:TestFederatedRoadmapMilestoneWithExternalMemberClosesThroughFederation
- R2 [satisfied] test:TestFederatedRoadmapMilestoneWithExternalMemberClosesThroughFederation
- R3 [satisfied] test:TestFederatedRoadmapMilestoneWithExternalMemberClosesThroughFederation
- R4 [satisfied] test:TestFederatedRoadmapMilestoneWithExternalMemberClosesThroughFederation

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
