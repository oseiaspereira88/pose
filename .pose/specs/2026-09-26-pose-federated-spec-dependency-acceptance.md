---
slug: pose-federated-spec-dependency-acceptance
status: in-progress
created_at: 2026-09-26
completed_at:
supersedes:
depends_on: pose-federated-roadmap-acceptance
priority: 0
components: pose-mcp
task_type: feature
delivers: contract:federated-spec-dependency-acceptance
---

# Spec: Federated acceptance for a spec's external dependencies

## 1. Intent

### Goal
Make review and closeout of a spec consult the same federated acceptance that
roadmaps and milestones use whenever the spec depends on an artifact owned by
another project.

### Business value
Harne8's adoption spec depends on `xref:proj.pose-dist/roadmap:pose-multirepo-foundation`.
With the consumer trust revoked, `closeout-check` on that spec reports zero
federated blockers, identical to the trusted baseline; only its milestone and
roadmap see the nine `consumer-trust-not-adopted` blockers. A spec can therefore
be reviewed and closed on top of a source its project no longer trusts.

### Constraints
Reuse the existing traversal, trust policy, sealed manifest and source proof;
do not build a second gate. Apply only to specs whose `depends_on` names a
qualified reference to another project, so specs without such edges keep a
byte-identical bundle payload. Local dependencies keep their existing
lifecycle rules. knowledge:multirepo-planning records the consumer finding.

### Non-goals
Do not traverse local dependencies from the spec scope, change roadmap or
milestone semantics, alter trust policy schema, or re-seal historic bundles.

## 2. Requirements

### Functional
- R1: A spec whose `depends_on` contains a qualified reference to another
  project seals a federated manifest over exactly those edges, built by the same
  traversal as roadmap acceptance, with the spec as coordinator.
- R2: Revoked or stale consumer trust, an unknown or unauthorized project,
  unavailable project roots, a source dependency that is not done or lacks an
  approved source review blocks `closeout-check` for that spec and makes an
  approved spec review stale; restoring the same trust restores freshness.
- R3: A spec without external qualified dependencies, and any store without a
  federated resolver, produces the same bundle payload and closeout state as
  before this change.
- R4: The coordinator revision of a spec manifest is the last commit that
  changed the spec file, so unrelated evidence commits do not stale its review.

### Security and compatibility
Keep resolution bounded and authorization-first. Report unavailable roots as a
blocker instead of failing the command, so a checkout without configured roots
cannot close the spec but can still read it. Historic bundles keep their sealed
contract; no evidence is rewritten.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/federated_acceptance.go`, `review_bundle.go`,
`review_closeout.go` and focused tests.

### Artifacts
- created: .pose/specs/2026-09-26-pose-federated-spec-dependency-acceptance.md
- modified: .pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/federated_acceptance_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- created: .pose/changelogs/unreleased/pose-federated-spec-dependency-acceptance.md

### Delivery targets
- contract:federated-spec-dependency-acceptance module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Extract the traversal behind `FederatedRoadmapAcceptance` into a function that
takes a coordinator identity and its edges. Add `FederatedSpecAcceptance`,
which returns no report when the spec has no external edge. Call it from bundle
preparation and closeout for spec scope, beside the roadmap call. Rollback is
a revert; bundles sealed with a spec manifest remain readable because the field
already exists in the payload schema.

## 4. Tasks

- [x] Reproduce the missing spec-scope blocker on the Harne8 consumer.
- [x] Write the negative and compatibility regressions before code.
- [x] Extract the traversal and add spec-scope acceptance.
- [ ] Run focused, race, vet and POSE module checks; rerun the consumer negative.
- [ ] Review and close with immutable evidence.

## 5. Decisions

The [federated acceptance ADR](../adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md)
governs this contract and is amended to name the spec scope. The trigger is the
qualified external edge itself: writing one is only possible after a project
adopts qualified references, so no separate adoption flag is added.

## 6. Validation

### Strategy
Risk: high, cross-project authorization. Unit fixtures build two independent
Git repositories; the consumer rerun proves the real pinned pair.

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Trusted external dependency composes, R1/R4 | `go test ./internal/pose -run TestFederatedSpecAcceptance -count=1` | Manifest present, ready, stable across an unrelated commit. |
| Revocation and restoration, R2 | `go test ./internal/pose -run TestFederatedSpecAcceptance -count=1` | Closeout blocked with `consumer-trust-not-adopted`; approved review stale, then fresh again. |
| No external edge or resolver, R3 | `go test ./internal/pose -run 'TestLegacyReviewBundlePayloadOmitsFederatedManifest|TestFederatedSpecAcceptance' -count=1` | No manifest; payload unchanged. |
| Existing federation contract | `go test -race ./internal/pose ./internal/cli -run Federated -count=1` | All pass without race. |
| Module matrix | `pose validate --strict --module pose-mcp --report` | Required checks pass. |
| Harne8 consumer negative | `closeout-check spec:harne8-multirepo-contract-adoption --json` with trust revoked, then restored | Blockers appear only while revoked. |

### Execution log
2026-09-26: on the Harne8 consumer pinned at `6dfbc1b`, emptying
`trusted_projects` left `spec:harne8-multirepo-contract-adoption` at zero
federated blockers while its milestone and roadmap rose from 3 to 12. The
three baseline blockers are local lifecycle states; all nine external edges
resolve done and trusted.

2026-09-26: the four new tests were added first. With `FederatedSpecAcceptance`
present but not called from bundle or closeout, the revocation and missing-roots
tests failed ("consumer spec bundle did not seal its federated dependency",
"missing roots did not block spec closeout"); the composition and
unrelated-commit tests passed. After wiring, the focused tests, `go vet`, the
race run of `./internal/pose ./internal/cli` for Federated/ReviewBundle/Closeout
and `go test ./...` passed. A candidate binary on the Harne8 pair, with a clean
`pose-dist` checkout, reported the adoption spec ready with zero federated
blockers, nine `consumer-trust-not-adopted` with trust revoked, and ready again
once restored. The same run against a dirty source checkout showed seven
`source-review-not-approved`: uncommitted engine edits stale the source reviews
that cover those files, so consumer negatives must run on a clean source.

### Requirement trace
- R1 [pending]
- R2 [pending]
- R3 [pending]
- R4 [pending]

## 7. Final Report

### Delivered scope
Pending implementation.

### Residual risks
Pending implementation.

### Follow-ups
- [open] Rerun the Harne8 adoption negative after this engine revision is
  pinned. (owner:@harne8-platform crit:high review:2026-10-03)
