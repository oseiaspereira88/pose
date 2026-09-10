---
slug: pose-only-the-signing-gate-is-read-live
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-reuse-is-sealed-signing-stays-live
priority: 0
components: pose-mcp
delivers:
task_type: feature
---

# Spec: The one live policy read is asserted, not remembered

## 1. Intent

### Goal
Fail if a second policy field is read live while judging a sealed bundle, or if
the signing requirement stops being read live.

### Business value
Everything a sealed bundle is judged by is now sealed with it, so that a setting
flipped today cannot re-judge a review recorded years ago. One gate is
deliberately not: `require_signed_attestations` is a bar rather than a
permission, and a bundle sealed before a project started requiring signatures
must not be permanently exempt.

That exception is one line of code and one paragraph of ADR. Sealing it by
symmetry with its neighbours would look like tidying, and would quietly exempt
every bundle already sealed from a security requirement. Adding a second live
read would reopen, without anyone deciding to, the retroactive judgement three
amendments have now closed.

Both are the kind of change that passes review by looking consistent.

### Constraints
- The assertion reads the function's own source, so it must name the function
  and fail loudly if it cannot find it.

### Non-goals
- Asserting anything about the legacy attempt path. It has no bundle, so live
  reads there are not the same question.

---

## 2. Requirements

### Functional
- R1: A policy field other than `require_signed_attestations` read while judging
  a sealed bundle shall fail, naming the field.
- R2: The signing requirement no longer being read live shall fail, naming the
  consequence.
- R3: A gate sealed into the bundle and never consulted shall fail.

### Non-functional
- The test fails, rather than silently passing, if the function it reads is
  renamed.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/live_policy_reads_test.go` — all of it

### Artifacts
- created: .pose/specs/2026-09-10-pose-only-the-signing-gate-is-read-live.md
- renamed: .pose/changelogs/unreleased/pose-only-the-signing-gate-is-read-live.md -> .pose/changelogs/v5.0.0/pose-only-the-signing-gate-is-read-live.md
- created: pose-mcp/internal/pose/live_policy_reads_test.go
- modified: .pose/specs/2026-09-10-pose-cross-version-guard-is-this-repositorys.md
- modified: .pose/specs/2026-09-10-pose-reuse-is-sealed-signing-stays-live.md

### Technical risks
- Reading source text is coarser than reading types, and a policy field reached
  through a local variable would not be seen. It catches the shape the two
  mistakes actually take — `policy.Field` at the point of decision — and the
  companion assertion catches a sealed gate that stops being consulted.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The live read is counted and named (R1, R2)
- [x] Increment 2: Every sealed gate is required to be consulted (R3)

### Validation
- [x] Both directions shown failing

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the exception could be documented, or asserted.
- Decision: asserted.
- Rationale: it is documented, in the ADR and at the call site, and this session
  has produced three cases where a documented invariant was undone by a change
  that looked like an improvement. The assertion costs one test and removes the
  need for the next reader to know the history.

---

## 6. Validation

### Strategy
Replace the live read with a sealed one and require the test to say what that
costs.

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
- Notes: replacing `policy.RequireSignedAttestations` with a sealed gate fails
  with "a bundle sealed before a project started requiring signatures is now
  permanently exempt from it". The companion assertion passes with all three
  gates consulted.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestOnlyTheSigningGateIsReadFromLivePolicy collects every `policy.Field` in the validator's own source and fails on any but the signing requirement, naming it and the remedy>
- R2 [satisfied] <the same test fails when the signing requirement is absent, naming what a bundle sealed earlier would be exempt from — verified by substituting a sealed gate for it>
- R3 [satisfied] <TestEverySealedGateIsConsulted requires all three sealed gates to appear in the validator, so a gate that seals and decides nothing fails>

### Known gaps
- The reader is a regular expression over source text. A field read through an
  intermediate variable would escape it.

---

## 7. Final Report

### Summary
The single deliberate exception to sealing is enforced rather than remembered.

### Follow-ups

- [open] Read the validator with the AST rather than a regular expression, so a policy field reached through a local variable is seen too — owner:unowned crit:low review:2027-03-10
