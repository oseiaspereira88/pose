---
slug: pose-reuse-is-sealed-signing-stays-live
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-bundle-findings-take-the-contract-the-legacy-path-had
priority: 0
components: pose-mcp
delivers:
task_type: refactor
---

# Spec: Criterion reuse is sealed; the signing requirement stays live

## 1. Intent

### Goal
Settle the last two policy fields a sealed bundle is judged by: seal
`allow_criterion_reuse`, and keep `require_signed_attestations` read from the
live policy on purpose.

### Business value
A survey of what verification reads live named six fields. Four turned out not
to be the question: `reviewer_independence` is already sealed in the plan,
`component_aware` is only read on the legacy attempt path where there is no
bundle, and `accepted_risk_severities` and `allow_approved_with_reservations`
were not read on the bundle path at all — a defect, fixed one spec earlier.

Two remain, and they look alike while pointing opposite ways.

Reuse carries a prior criterion disposition into this attestation. Whether that
was permitted is a property of the review that happened: read live, a project
enabling reuse today retroactively legitimises an attestation that reused a
criterion when its own policy forbade it.

The signing requirement is a bar, not a permission. A project that starts
requiring signed attestations is raising it, and a bundle sealed before that must
not be permanently exempt — sealing it would mean the new requirement applies
only to future work, which is the opposite of what adopting it means.

### Constraints
- The exception must be visible where the code makes it, not only in an ADR.

### Non-goals
- Sealing the signing requirement for consistency. Consistency here would freeze
  a security requirement at the moment it was weakest.

---

## 2. Requirements

### Functional
- R1: Criterion reuse shall be permitted or refused by the bundle's sealed
  gates, not by the policy on disk.
- R2: A bundle sealed before the gate existed shall refuse reuse, which is the
  conservative reading.
- R3: The signing requirement shall continue to be read from the live policy, so
  raising it reaches a bundle sealed before it.

### Non-functional
- The one remaining live read is annotated at the call site with why.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — the gate, the seal, and the
  annotated exception

### Artifacts
- created: .pose/specs/2026-09-10-pose-reuse-is-sealed-signing-stays-live.md
- renamed: .pose/changelogs/unreleased/pose-reuse-is-sealed-signing-stays-live.md -> .pose/changelogs/v5.0.0/pose-reuse-is-sealed-signing-stays-live.md
- created: pose-mcp/internal/pose/reuse_and_signing_gates_test.go
- modified: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: .pose/specs/2026-09-09-pose-bundles-seal-the-contracts-that-govern-them.md
- modified: .pose/specs/2026-09-10-pose-bundle-findings-take-the-contract-the-legacy-path-had.md
- modified: pose-mcp/internal/pose/bundle_finding_contract_test.go
- modified: .pose/indexes/delivery-integrity.json

### Technical risks
- This change also carries the pointer correction to the gates field the
  previous spec introduced, because the two are the same lines of the same
  function and cannot be committed apart. The previous spec's own record and
  test are claimed here as modified rather than left undeclared.
- A bundle sealed before this gate refuses reuse where an instance with reuse
  enabled previously allowed it. That is the intended reading and it is narrow:
  reuse is opt-in, and an attestation that used it is re-recordable.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Reuse read from the sealed gates (R1, R2)
- [x] Increment 2: The signing exception annotated where it is made (R3)

### Validation
- [x] Reuse shown ignoring a policy that would permit it

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the survey asked for a decision per field, and the measurement
  reduced six to two.
- Decision: seal reuse, keep signing live, and correct the survey's own
  follow-up rather than leave it asking for six.
- Rationale: a backlog item that asks for more than the question contains costs
  whoever picks it up the same measurement twice. Four of the six were already
  answered by the code; saying so is cheaper than re-deriving it.

---

## 6. Validation

### Strategy
Assert reuse against a policy that permits it and a bundle that does not, and
assert the signing requirement reaching a bundle that predates it.

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
- Date: 2026-09-10
- Environment: local, Go 1.26
- Notes: with `allow_criterion_reuse: true` on disk and a bundle sealed without
  the gate, reuse is refused — the case sealing exists for. With the gate sealed,
  it is permitted. An unsigned attestation is refused under a policy requiring
  signatures even though the bundle predates the requirement. After this change
  `RequireSignedAttestations` is the only live policy read left in
  `validateBundleAttestationWith`.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <validateBundleAttestationWith reads bundle.Payload.Gates.AllowCriterionReuse; TestCriterionReuseTakesTheSealedGate covers both answers>
- R2 [satisfied] <TestReuseIgnoresThePolicyOnDisk writes a policy enabling reuse and requires a gateless bundle to refuse it anyway>
- R3 [satisfied] <TestSigningRequirementStaysLive requires an unsigned attestation against a gateless bundle to be refused under a policy that requires signatures>

### Known gaps
- The signing exception is one line of code and one paragraph of ADR. Nothing
  fails if a later change seals it by symmetry with the others.

---

## 7. Final Report

### Summary
Every gate a sealed bundle is judged by is sealed with it except the signing
requirement, and that exception is stated where it is made.

### Follow-ups

- [done] Assert that `require_signed_attestations` is the only live policy read left in the bundle validator, so sealing it by symmetry with the others fails rather than passing quietly. Done in `pose-only-the-signing-gate-is-read-live`: the validator's own source is read and any policy field but that one fails, naming the field; the signing requirement going missing fails too, naming what a bundle sealed earlier would be exempt from. A companion assertion requires every sealed gate to be consulted. (owner:unowned crit:medium review:2026-12-10)
