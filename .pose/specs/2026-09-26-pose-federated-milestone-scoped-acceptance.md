---
slug: pose-federated-milestone-scoped-acceptance
status: in-progress
created_at: 2026-09-26
supersedes:
depends_on: pose-federated-roadmap-acceptance
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:federated-milestone-scoped-acceptance
---

# Spec: Milestone-scoped federated acceptance

## 1. Intent

### Goal
Make review and closeout of a milestone consult federated acceptance over that
milestone's own edges, not over every member of its roadmap.

### Business value
In Harne8, `closeout-check milestone:harne8-multirepo-consistency/adoption`
reports `xref:proj.harne8/spec:harne8-abm-spec-authority-reconciliation:not-done`.
That spec belongs to the next milestone, `abm-reconciliation`, which declares
`after: adoption`. Every milestone before the last therefore stays open until
the whole roadmap is done, so the ordering the roadmap declares cannot be
closed in that order. Local milestone closeout already considers only the
milestone's own children.

### Constraints
Reuse the existing traversal, trust policy and source proof. A milestone keeps
the roadmap's own `depends_on` and `consumes` edges, because they gate the
whole program. Bundles sealed before this change keep their sealed semantics
(R8 of `pose-federated-roadmap-acceptance`): a milestone bundle sealed with its
roadmap as federated coordinator stays fresh while that snapshot is
unchanged. Do not change `validation-matrix.json`, which is an input of every
sealed bundle and of the trust contract digest. Harne8's multirepo planning
handoff records the consumer plan that this blocks.

### Non-goals
Do not change roadmap-scope acceptance, spec-scope acceptance, the trust
policy schema, or re-seal historic bundles.

## 2. Requirements

### Functional
- R1: A milestone's federated manifest is built by the same traversal as
  roadmap acceptance, with the milestone as coordinator, over the roadmap's
  `depends_on` and `consumes` plus that milestone's `after`, `specs` and
  `consumes`. Members of other milestones are absent.
- R2: Milestone closeout reports blockers only from that manifest; a later
  milestone and the roadmap keep reporting their own open members, and revoked
  trust in an edge the milestone holds still blocks it.
- R3: A milestone bundle sealed with the roadmap-wide manifest stays fresh and
  approved while the legacy preparation reproduces its digest, and goes stale
  on revocation or source change like any other seal.
- R4: New milestone seals carry the milestone-scoped manifest and are the
  current bundle once attested. An unknown milestone fails instead of composing.

### Security and compatibility
The legacy snapshot contains every edge of the milestone-scoped manifest, so
accepting it never widens what a milestone review is trusted for. Resolution
stays bounded and authorization-first. No evidence or sealed bundle is
rewritten.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/federated_acceptance.go`, `review_bundle.go`,
`review_closeout.go` and focused tests.

### Artifacts
- created: .pose/specs/2026-09-26-pose-federated-milestone-scoped-acceptance.md
- modified: .pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/federated_acceptance_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- created: .pose/changelogs/unreleased/pose-federated-milestone-scoped-acceptance.md

### Delivery targets
- contract:federated-milestone-scoped-acceptance module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Add `FederatedMilestoneAcceptance`, which selects one milestone from the
roadmap and reuses `collectFederatedRoadmapEdges`. Closeout and bundle
preparation call it for milestone scope. `matchSealedReviewBundle` replaces the
digest match in `CurrentReviewBundle` and `VerifyReviewBundle`; when no bundle
matches and a sealed milestone bundle names a roadmap coordinator, it prepares
the legacy variant once and matches only those bundles. Rollback is a revert;
milestone bundles sealed by the new engine then read as superseded and are
resealed.

## 4. Tasks

- [x] Reproduce the cross-milestone blocker on the Harne8 consumer.
- [x] Write the regressions and prove they fail without the new wiring.
- [x] Add milestone-scoped acceptance and legacy seal matching.
- [ ] Run focused, race, vet and POSE module checks; rerun the consumer.
- [ ] Review and close with immutable evidence.

## 5. Decisions

The [federated acceptance ADR](../adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md)
is amended to name the milestone scope. Legacy matching is chosen over
resealing: 36 sealed bundles of the four `pose-multirepo-foundation` milestones
carry the roadmap coordinator, and Harne8's source proof verifies them through
`VerifyReviewBundle`. The regressions are named `TestFederatedRoadmapMilestone…`
so the existing `federated-roadmap-acceptance-integration` check selects them
without a matrix change.

## 6. Validation

### Strategy
Risk: high, cross-project authorization and sealed review freshness. Unit
fixtures build two independent Git repositories; the consumer rerun proves the
real pinned pair.

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Milestone manifest and closeout, R1/R2/R4 | matrix check `federated-roadmap-acceptance-integration` | Earlier milestone ready; later milestone and roadmap blocked by the open member; revocation blocks. |
| Legacy and new seals, R3/R4 | `go test ./internal/pose -run TestFederatedRoadmapMilestone -count=1` | Legacy seal fresh, stale on revocation; new seal carries a milestone coordinator. |
| Existing federation contract | `go test -race ./internal/pose ./internal/cli -run 'Federated|ReviewBundle|Closeout' -count=1` | All pass without race. |
| Module matrix | `pose validate --strict --module pose-mcp --report` | Required checks pass. |
| Harne8 consumer | `closeout-check milestone:harne8-multirepo-consistency/adoption --json` | No blocker from the `abm-reconciliation` member. |

### Execution log
2026-09-26: on Harne8 pinned at `e36c41f`, `closeout-check` of the adoption
milestone reported `xref:proj.harne8/spec:harne8-abm-spec-authority-reconciliation:not-done`
beside the missing review. The cause is `GetCloseoutState` and
`PrepareReviewBundle` calling `FederatedRoadmapAcceptance(scope.Roadmap)` for
milestone scope.

2026-09-26: with the new functions present but closeout and preparation wired
back to the roadmap manifest, both regressions failed ("earlier milestone
closeout carries a later member blocker", "new milestone seal did not carry
its own manifest"). With legacy matching disabled, the legacy seal read as
superseded ("legacy milestone seal lost freshness under the new engine").
With the full change, the focused tests, `go vet`, the race run of
`./internal/pose ./internal/cli` for Federated/ReviewBundle/Closeout and
`go test ./...` passed.

### Requirement trace
- R1 [satisfied] test:TestFederatedRoadmapMilestoneAcceptanceIgnoresLaterMilestones
- R2 [satisfied] test:TestFederatedRoadmapMilestoneAcceptanceIgnoresLaterMilestones
- R3 [satisfied] test:TestFederatedRoadmapMilestoneBundleSealsOwnManifestAndKeepsLegacySeals
- R4 [satisfied] test:TestFederatedRoadmapMilestoneBundleSealsOwnManifestAndKeepsLegacySeals test:TestFederatedRoadmapMilestoneAcceptanceIgnoresLaterMilestones

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
