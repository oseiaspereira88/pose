---
slug: pose-bundles-seal-the-contracts-that-govern-them
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-contract-adoption-registry
priority: 0
components: pose-mcp
delivers:
---

# Spec: A review bundle names the contracts that govern it

## 1. Intent

### Goal
Seal, into the bundle, which governance contracts were in force when it was
sealed, and judge it by that instead of by a date in today's policy.

### Business value
The exemption that keeps a finished scope's approval when a contract arrives
after it compares the review's timestamp against `contract_adoptions` in the
live policy. Three things follow from that:

The date is editable and re-read on every verification. Moving
`evidence_vocabulary_reconciled_at` forward retroactively exempts more
historical closeouts, so an immutable bundle is judged by today's configuration.
That is the objection this repository already accepted when it sealed
`selected_profiles` rather than reading profile selection live — the same
argument, applied to the other thing verification still read from policy.

Every new contract needs a new dated marker, and a legacy field beside it. The
registry consolidated the mechanism; the dating kept the per-contract encoding.

And the exemption is by time rather than by what the bundle contains. A bundle
sealed by an engine that never knew a contract is indistinguishable, to the
gate, from one sealed after adoption by an instance whose date was edited.

### Constraints
- The 483 bundles already sealed carry no such field and cannot be back-filled.
  Rewriting an append-only immutable artifact is what sealing exists to prevent,
  and re-sealing would break the attestations bound to their digests.
- The legacy attempt path has no bundle at all.

### Non-goals
- The other policy fields verification reads live: `allow_criterion_reuse`,
  `require_signed_attestations`, `reviewer_independence`,
  `accepted_risk_severities`, `allow_approved_with_reservations`,
  `component_aware`. Each needs its own answer, and at least one probably should
  stay live.
- Removing the dated rule. It is demoted, not deleted, and cannot be deleted
  while an unstamped bundle is reachable.

---

## 2. Requirements

### Functional
- R1: The sealed payload shall carry the ids of the governance contracts this
  engine knows, and the field shall be inside the digest.
- R2: What is sealed shall be the registry, so a contract added there governs
  every bundle sealed afterwards with no date to stamp.
- R3: A bundle that names a contract shall be held to it regardless of the
  adoption date, including a date moved forward afterwards.
- R4: A bundle that names contracts but not this one shall be exempt from it on
  its own terms, independent of the policy, and still only for a finished scope.
- R5: A bundle carrying no list shall keep the dated reading, and the legacy
  attempt path shall keep it too.

### Non-functional
- A bundle sealed before this change keeps its digest and its verdict.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — the payload field, the seal, the
  lookup
- `pose-mcp/internal/pose/review_closeout.go` — `bundleContractExempt`, used
  where a sealed bundle exists

### Artifacts
- created: .pose/specs/2026-09-09-pose-bundles-seal-the-contracts-that-govern-them.md
- renamed: .pose/changelogs/unreleased/pose-bundles-seal-the-contracts-that-govern-them.md -> .pose/changelogs/v4.0.0/pose-bundles-seal-the-contracts-that-govern-them.md
- created: pose-mcp/internal/pose/governing_contracts_test.go
- modified: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: .pose/specs/2026-09-08-pose-contract-adoption-registry.md

### Technical risks
- `BundleGovernedBy` returns two booleans because "this bundle says the contract
  did not govern it" and "this bundle says nothing" are different facts.
  Collapsing them would make all 483 legacy bundles read as ungoverned by
  everything, which is a blanket exemption rather than a reading.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The payload names the contracts, inside the digest (R1, R2)
- [x] Increment 2: Verification asks the bundle where one says (R3, R4)
- [x] Increment 3: An unstamped bundle keeps the dated reading (R5)

### Validation
- [x] The assertions shown failing when the dated path is forced

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: four ways to make an immutable bundle stop depending on an editable
  date, put to the maintainer with the measurements.
- Decision: seal the contract ids in the payload; demote the dated rule to the
  reading of an unstamped bundle. Recorded as an amendment to the sealed review
  bundles ADR.
- Rationale: stamping outside the digest would leave the value editable, which is
  the defect in a new place. Deriving the contracts from the bundle's own content
  works retroactively but infers what should be declared, and does not generalise
  to a contract that leaves no trace. Sealing the ids is the only option that is
  both immutable and explicit; the legacy bundles keep the dated reading, said
  out loud rather than inferred.

### Decision 2
- Date: 2026-09-09
- Context: what "in force" means at seal time.
- Decision: every contract the engine knows.
- Rationale: adoption dates are in the past or absent by the time a bundle is
  sealed, so the set in force is the registry. It also gives the property the
  date could not: a bundle sealed by an engine that did not know a contract never
  lists it, permanently, and no later edit reaches it.

### Decision 3
- Date: 2026-09-09
- Context: the first version of this spec's central test passed against both the
  new mechanism and the old one.
- Decision: the fixture asserts its own premise — that the scope is done and that
  the dated rule *would* grant the exemption — before asserting that the stamped
  bundle refuses it.
- Rationale: `scopeLifecycleDone` fails on an empty instance, so the dated path
  returned false for a reason unrelated to the contract and the assertion held
  for the wrong reason. Forcing the old mechanism did not fail the test, which is
  how it was caught. That check is now the first thing the fixture does.

---

## 6. Validation

### Strategy
Assert both readings on a fixture whose premise is asserted first, and require
the assertions to fail when the dated path is forced.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: forcing `bundleContractExempt` down the dated path fails both stamped
  assertions. The first version of the fixture did not fail under that mutation,
  because the scope was not done and the dated rule declined for an unrelated
  reason; the fixture now refuses to run unless the dated rule would exempt it.
  `pose check --strict` passes, and the 483 bundles already sealed are untouched.

### Results summary
- Successes: R1 through R5 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <ReviewBundlePayload.GoverningContracts is populated at seal from governingContractsAtSeal; TestGoverningContractsAreInsideTheDigest asserts the field changes the payload digest, which is what makes it sealed>
- R2 [satisfied] <TestSealedContractsAreTheRegistry compares what is sealed with ReviewContracts() and fails on any difference, and fails if the sealed set is empty>
- R3 [satisfied] <TestAStampedBundleIgnoresTheAdoptionDate uses an adoption date of 2099-01-01, which the fixture first proves would exempt the review, and requires the stamped bundle to be held anyway>
- R4 [satisfied] <TestAStampedBundleWithoutTheContractIsExemptOnItsOwnTerms requires the exemption with the date present and with the policy empty, so the bundle and not the date is what grants it>
- R5 [satisfied] <TestAnUnstampedBundleStillReadsByTheDate requires the dated exemption with a date recorded and no exemption without one; the legacy attempt path still calls reviewCompletedBeforeContract directly, since it has no bundle>

### Known gaps
- The dated rule remains, for unstamped bundles and for legacy attempts. Nothing
  reports how many unstamped bundles are still reachable, so there is no signal
  for when it can be removed.
- Verification still reads six policy fields live. This spec covers the
  contracts only.

---

## 7. Final Report

### Summary
A bundle says which contracts governed it, and no later edit to a date can
change that answer.

### Follow-ups

- [done] Decide, per field, whether verification should read `allow_criterion_reuse`, `require_signed_attestations`, `reviewer_independence`, `accepted_risk_severities`, `allow_approved_with_reservations` and `component_aware` from the sealed bundle or from live policy — tightening a signing requirement probably should reach an old bundle, and reusing a criterion probably should not. Measured rather than decided six times: `reviewer_independence` is already sealed in the plan and read live only on the legacy attempt path; `component_aware` is legacy-only, where there is no bundle to seal into; `accepted_risk_severities` and `allow_approved_with_reservations` were not read on the bundle path at all, which was a defect fixed in `pose-bundle-findings-take-the-contract-the-legacy-path-had`. The two real ones are settled in `pose-reuse-is-sealed-signing-stays-live`: reuse sealed, signing live on purpose. (owner:unowned crit:medium review:2026-12-09)
- [open] Report how many sealed bundles carry no governing_contracts, so the dated rule has a removal signal instead of living forever by default (owner:unowned crit:low review:2027-03-09)
