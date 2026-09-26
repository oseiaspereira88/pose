---
slug: pose-federated-carried-forward-proof
status: in-progress
created_at: 2026-09-26
completed_at:
supersedes:
depends_on: pose-federated-roadmap-acceptance
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:federated-carried-forward-proof
---

# Spec: Federated proof for current carried-forward evidence

## 1. Intent

### Goal
Accept current, post-subject validation evidence from an approved source review
when federating a closed delivery.

### Business value
Harne8 pins a reviewed POSE outcome, but federated acceptance rejects all four
source specs because their passing checks ran after the implementation commit.

### Constraints
Preserve explicit consumer trust, exact source revision, current approved source
review and delivery profile requirements. The source review classifies these
checks as `carried-forward` and current by provenance. The consumer adoption
remains a separate authority. knowledge:multirepo-review-continuation records
the earlier handoff.

### Non-goals
Do not accept unknown evidence, an earlier check that could not observe the
subject, stale reviews, changed trust pins or status-only delivery claims.

## 2. Requirements

### Functional
- R1: A source spec with a fresh approved review may satisfy a delivery class
  using passing `carried-forward` evidence only when its Git head is a descendant
  of the reviewed subject head and an ancestor of the pinned source revision.
- R2: Evidence without a Git head, before the subject head, outside the pinned
  ancestry, failed, or from an unapproved/stale source review must not satisfy
  a required class.
- R3: The federated manifest must expose the accepted evidence and remain
  sensitive to trust revocation and source revision changes.

### Security and compatibility
Use only committed Git ancestry and the sealed source bundle. Keep the existing
`observed` case and historical bundle semantics. Do not mutate source evidence.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/federated_acceptance.go` and its focused tests.

### Artifacts
- created: .pose/specs/2026-09-26-pose-federated-carried-forward-proof.md
- modified: .pose/adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: pose-mcp/internal/pose/federated_acceptance_test.go
- created: .pose/reports/2026-09-26-federated-carried-forward-proof.md

### Delivery targets
- contract:federated-carried-forward-proof module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Use one Git ancestry check per distinct evidence head. Accept the carried value
only inside an already fresh and approved source bundle, with passing outcome,
current provenance and the required target module/class. Revert this fix and
the consumer trust pin together if a negative gate regresses; preserve sealed
bundles and never rewrite historical attestations.

## 4. Tasks

- [x] Reproduce the consumer blocker and isolate the `observed` filter.
- [x] Define the positive and negative regression scenarios before code.
- [x] Implement the minimal ancestry guard and manifest projection.
- [x] Run focused, race, vet and POSE module checks.
- [ ] Review and close with immutable evidence.

## 5. Decisions

The existing [federated acceptance ADR](../adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md)
governs this contract. A passing post-subject check remains valid only when the
source reviewer approved its current provenance; temporal ordering is an
additional guard. A pre-subject result cannot prove the implemented change.

## 6. Validation

### Strategy
Risk: high, cross-project evidence and authorization. The focused unit test
covers the source proof boundary, the integration test covers the federated
consumer, and the existing matrix checks the CLI/MCP contract.

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Post-subject carried evidence, R1/R3 | `go test ./internal/pose -run TestFederatedAcceptanceCarriedForwardProof -count=1` | A reviewed pinned source composes and the manifest names the accepted check. |
| Pre-subject, unknown, failed and out-of-pin evidence, R2 | `go test ./internal/pose -run TestFederatedAcceptanceCarriedForwardProof -count=1` | Each variant reports missing source delivery evidence. |
| Revoked trust and changed source revision, R3 | `go test ./internal/pose -run 'TestFederatedAcceptanceNegative|TestFederatedAcceptanceComposesReviewedSiblingRoadmap' -count=1` | Existing negative blockers remain. |
| Go contract and race | `go test -race ./internal/pose -run 'TestFederatedAcceptance' -count=1` | All focused tests pass without race. |
| Module matrix and vet | `pose validate --strict --module pose-mcp --report` | Required POSE checks pass. |
| Harne8 consumer proof | `pose roadmap-check harne8-multirepo-consistency --strict` with the adopted roots | No `source-delivery-evidence-missing` blocker for the pinned source; local composition may remain open. |

### Execution log
2026-09-26: Harne8's trusted federated policy resolved source reviews at
`c19ff2d` but reported `source-delivery-evidence-missing` for four specs.
Their sealed bundles carry passing current results marked `carried-forward`
because validation at `a8a116e` followed each subject commit. The source
`review verify` is fresh and approved. The new regression failed before the
fix, then passed for the post-subject case; earlier, unknown and outside-pin
heads stayed blocked. Focused CLI/pose tests, the race run and vet passed. The
first `pose-mcp` matrix run passed 25/25 during implementation; repeat it
after commit for canonical evidence. The second module run passed 25/25 after
the negative assertions. At commit `adaa497`, the full matrix passed 28/28
outside the sandbox; its sandboxed run failed nine MCP checks because local
`httptest` sockets were forbidden. `pose assess tech-debt` found zero markers;
`pose assess integrate` retained 56 existing inventory gaps among 57
contracts. Root `pose check --strict` found one unrelated missing knowledge
artifact claimed by `pose-scaffold-self-referential-policy-fix`; this spec
does not change that artifact.

### Requirement trace
- R1 [deferred-integration: regression and consumer proof pending] report:.pose/reports/2026-09-26-federated-carried-forward-proof.md
- R2 [deferred-integration: negative regression pending] report:.pose/reports/2026-09-26-federated-carried-forward-proof.md
- R3 [deferred-integration: manifest and trust gates pending] report:.pose/reports/2026-09-26-federated-carried-forward-proof.md

## 7. Final Report

### Delivered scope
Implementation pending.

### Residual risks
Do not activate consumer trust if the source review, pin or validation
provenance changes before the consumer check.

### Follow-ups
- [open] Recheck Harne8's federation pin after the source fix is reviewed and
  adopted. (owner:@harne8-platform crit:high review:2026-09-27)
