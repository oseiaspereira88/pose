---
slug: pose-roadmap-gate-scopes-milestones-and-external-members
status: done
created_at: 2026-09-26
supersedes:
depends_on: pose-federated-milestone-scoped-acceptance
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:roadmap-gate-scopes
completed_at: 2026-09-27
---

# Spec: Roadmap gate scoped to milestones and external members

## 1. Intent

### Goal
Make `pose close milestone:<roadmap>/<id>` gate on that milestone alone, and
make `roadmap-check` judge members owned by another project through
federated acceptance instead of local closeout.

### Business value
In Harne8, `closeout-check milestone:harne8-multirepo-consistency/adoption`
is terminal with a fresh approved review, yet `pose close` refuses it with
`member spec is not terminal: harne8-abm-spec-authority-reconciliation`, a
member of the next milestone: the close path runs the whole
`roadmap-check --strict`, cut criteria included. In a rehearsal pointing
`pose-abm-foundation` members at `xref:proj.pose-dist/spec:*`, every such
member produced `member spec is not terminal: xref:...`, because the check
reads each member through local closeout, which cannot resolve another
project's spec. Both keep an ordered or federated roadmap from closing in the
order it declares.

### Constraints
Reuse `FederatedMilestoneAcceptance` and the existing roadmap criteria. An
external member keeps every federated blocker (status, trust, source review,
evidence); only the local closeout lookup is skipped for it. The roadmap gate
keeps its cut criteria and every member. Do not change validation-matrix.json.

### Non-goals
Do not change milestone lifecycle persistence, roadmap cut-criteria semantics,
review eligibility of `pose close`, or the federated traversal.

## 2. Requirements

### Functional
- R1: The milestone gate of `pose close` evaluates the milestone's federated
  acceptance and only its own members, without the roadmap cut criteria; an
  unknown milestone fails.
- R2: `roadmap-check` keeps reporting every open member of every milestone and
  every failing cut criterion.
- R3: A member qualified with another project is not looked up in local
  closeout and stays gated by federated acceptance; a member qualified with
  the local project resolves by its slug.

### Security and compatibility
Fail closed: an external member absent from federated acceptance cannot be
missed, because ownership edges come from the same milestone members. Output
of `roadmap-check` keeps its JSON shape.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/cli/surface_check.go` (shared `roadmapGate`),
`review_closeout.go` (milestone close) and focused tests.

### Artifacts
- created: .pose/specs/2026-09-26-pose-roadmap-gate-scopes-milestones-and-external-members.md
- modified: .pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/roadmap_gate_scope_test.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- created: .pose/changelogs/unreleased/pose-roadmap-gate-scopes-milestones-and-external-members.md

### Delivery targets
- contract:roadmap-gate-scopes module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Extract the body of `cmdRoadmapCheck` into `roadmapGate(root, slug, milestone)`.
With a milestone it selects milestone acceptance, drops cut criteria and
iterates that milestone's members. The member loop skips members qualified
with another project. Milestone close calls it instead of `cmdRoadmapCheck`.
Rollback is a revert.

## 4. Tasks

- [x] Reproduce both refusals on the Harne8 consumer.
- [x] Write the regressions and prove they fail under the previous behaviour.
- [x] Extract the shared gate and scope it.
- [x] Run module checks, review and close.

## 5. Decisions

Amends the [federated acceptance ADR](../adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md).
`pose close` requires an approved review before any gate, so the regressions
call `roadmapGate` directly; the close wiring is one call, measured on the
Harne8 adoption milestone after pinning.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Milestone scope, R1/R2 | matrix check `delivery-integration` | Earlier milestone passes; later milestone blocked by its member; roadmap keeps member and cut criterion. |
| External members, R3 | `go test ./internal/cli -run TestRoadmapCheckGateJudgesExternalMembersByFederation -count=1` | No local lookup for the external member; federated blocker remains. |
| Module matrix | `pose validate --strict --module pose-mcp --report` | Required checks pass. |
| Harne8 consumer | `pose close milestone:harne8-multirepo-consistency/adoption` | Milestone closeout verified. |

### Execution log
2026-09-26: with the previous behaviour simulated (milestone ignored, external
members looked up locally), both regressions failed; with the change they
pass, together with `go vet ./...`, the race run of `internal/cli` for
RoadmapGate/RoadmapCheck/Federated/Closeout and `go test ./...`. The
print-site ratchet dropped two direct prints in `surface_check.go` and one in
`review_closeout.go`, and its baseline was lowered accordingly.
At `6ac78ab` the full matrix passed 29/29 with a candidate binary from that
commit; `artifact-check` reported 7 claims and 7 observed paths. Bundle
`rvb-1c38a09b5be428e4` was approved by attestation `rva-3f31a82e56905a35`,
recorded by the agent under explicit authorization from the user to
self-attest. The commit superseded the foundation reviews again, since
`surface_check.go` and `review_closeout.go` are artifacts of those specs; the
eight specs, four milestones and roadmap were resealed and reattested.

### Requirement trace
- R1 [satisfied] test:TestRoadmapCheckGateMilestoneIgnoresLaterMembersAndCutCriteria
- R2 [satisfied] test:TestRoadmapCheckGateMilestoneIgnoresLaterMembersAndCutCriteria
- R3 [satisfied] test:TestRoadmapCheckGateJudgesExternalMembersByFederation

## 7. Final Report

### Delivered scope
Milestone close gates on the milestone alone, and roadmap-check leaves
members owned by another project to federated acceptance.

### Residual risks
`pose close` still requires an approved review before any gate, so the close
wiring is covered by the consumer measurement rather than a unit test.

### Follow-ups
- [done] Pin this engine revision in Harne8 and close `milestone:harne8-multirepo-consistency/adoption` through `pose close`. Resolved, reconciled 2026-10-04 (spec pose-open-backlog-reconciliation): Harne8 roadmap harne8-multirepo-consistency is status done.
