---
slug: pose-bundle-findings-take-the-contract-the-legacy-path-had
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-bundles-seal-the-contracts-that-govern-them
priority: 0
components: pose-mcp
delivers:
task_type: bugfix
---

# Spec: The bundle path enforces the finding contract it never had

## 1. Intent

### Goal
Apply, when validating an attestation against a sealed bundle, the same finding
contract the legacy attempt path always applied — with the two policy-dependent
parts read from the bundle rather than from today's policy.

### Business value
`validateBundleAttestationWith` blocked a finding only when its disposition was
`open` or `changes-requested`. Measured against the legacy path, three cases
passed with no blocker at all:

- an accepted risk of severity `critical` with no owner, no rationale and no
  review date;
- a disposition the engine does not know;
- a finding with neither severity nor action.

And two policy settings were not consulted on this path at all.
`allow_approved_with_reservations` was ignored — the check is a literal
`att.Decision != "approved"`, so a project that enabled reservations found them
refused anyway. `accepted_risk_severities` was ignored, so the list a project
curated decided nothing.

A project that adopted review bundles therefore lost a gate it had before, and
two settings stopped taking effect, with nothing reporting either. Review
bundles exist to make review more rigorous; on this axis they made it less.

This was found by reading the two paths against each other while surveying which
policy fields verification reads live — not from a report, which is the point:
nothing would have reported it.

### Constraints
- No bundle already sealed may change verdict.

### Non-goals
- Sealing the structural half of the contract. A finding without a severity is
  incomplete under any policy, and sealing it would imply a project could
  configure it away.

---

## 2. Requirements

### Functional
- R1: The bundle path shall refuse an incomplete accepted risk, an unknown
  disposition, and a finding lacking severity or action.
- R2: Which risk severities may be accepted shall come from the bundle's sealed
  gates.
- R3: Whether an approved-with-reservations decision may close a scope shall
  come from the bundle's sealed gates.
- R4: A bundle carrying no gates shall read as reservations refused and no
  severity accepted, which is what this path already did.

### Non-functional
- The structural half applies to every bundle, sealed before or after.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — the payload's `gates`, the seal,
  and `validateBundleAttestationWith`

### Artifacts
- created: .pose/specs/2026-09-10-pose-bundle-findings-take-the-contract-the-legacy-path-had.md
- renamed: .pose/changelogs/unreleased/pose-bundle-findings-take-the-contract-the-legacy-path-had.md -> .pose/changelogs/v5.0.0/pose-bundle-findings-take-the-contract-the-legacy-path-had.md
- created: pose-mcp/internal/pose/bundle_finding_contract_test.go
- modified: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- modified: pose-mcp/internal/pose/review_bundle.go

### Technical risks
- `omitempty` does nothing for a struct value. The first version of `Gates` was
  one, so every payload serialised `"gates":{}` and the digest of all 483
  bundles already sealed changed — `pose check --strict` reported the mismatch
  across dozens of closed scopes. It is a pointer now: nil means the bundle
  predates the field, and a pointer to an empty struct means it was sealed with
  the defaults, which are different facts.
- Applying the structural contract to bundles sealed earlier could invalidate a
  historical closeout. Measured before deciding: the 469 attestations recorded
  in this repository carry no findings at all and none is
  approved-with-reservations, so nothing is re-judged here. A repository with
  such findings would see them, correctly.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The finding contract on the bundle path (R1)
- [x] Increment 2: The two policy gates sealed and read from the bundle (R2, R3, R4)

### Validation
- [x] The tests shown failing against the gate as it was

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the two settings could be read live at verification or sealed.
- Decision: sealed.
- Rationale: read live, flipping `allow_approved_with_reservations` today would
  approve a closeout recorded years ago — the objection this ADR accepted one
  amendment earlier for the governance contracts, and the same one applies here.

### Decision 2
- Date: 2026-09-10
- Context: whether to seal the whole contract or only the policy-dependent part.
- Decision: only the policy-dependent part.
- Rationale: a finding without a severity is incomplete under any policy.
  Sealing that would imply a project could configure it away, which is not a
  setting anyone should have.

### Decision 3
- Date: 2026-09-10
- Context: whether the structural half should apply to bundles sealed before it
  existed.
- Decision: yes.
- Rationale: it is not a contract those bundles were judged under, it is a check
  that was missing. Grandfathering it would preserve a defect. The retroactive
  cost was measured first and is zero here — 469 attestations, no findings.

---

## 6. Validation

### Strategy
Assert each case the legacy path refused and this one did not, and require the
assertions to fail against the gate as it was.

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
- Notes: the first version of the sealed field was a struct value, and
  `pose check --strict` failed across dozens of closed scopes with digest
  mismatches — `omitempty` is a no-op on a struct, so every sealed bundle's
  payload gained `"gates":{}`. With a pointer the gate is clean again and the
  483 recorded bundles keep their digests. Restoring the previous two-case loop
  fails six of the new assertions.
  Before the fix, a probe of the three cases returned `blockers=[]` for all
  three. A scan of `.pose/review-attestations/` found 469 attestations and zero
  findings, which is why the structural half could apply retroactively without
  re-judging anything.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestBundlePathRefusesTheFindingsTheLegacyPathAlwaysDid covers the incomplete accepted risk, the unknown disposition, the finding lacking severity or action, and the open finding that was the only case it ever caught>
- R2 [satisfied] <TestBundlePathAcceptsACompleteAcceptedRisk passes a complete risk whose severity the sealed gates accept and refuses the same finding at a severity they do not>
- R3 [satisfied] <TestApprovedWithReservationsTakesTheSealedGate refuses reservations under empty gates and accepts them under gates that allow them>
- R4 [satisfied] <TestAnUnsealedGateIsTheConservativeReading requires empty gates to refuse both reservations and an accepted risk>

### Known gaps
- Nothing reports that a bundle carries no gates, so a repository cannot tell
  which of its bundles were sealed before the field existed.

---

## 7. Final Report

### Summary
Adopting review bundles no longer costs a project the finding gate it had, and
the two settings that govern it are frozen with the bundle.

### Follow-ups

- [open] Report how many sealed bundles carry no `gates`, alongside the same count for `governing_contracts` — both are pre-field bundles reading by a fallback, and neither has a signal for when the fallback can go (owner:unowned crit:low review:2027-03-10)
