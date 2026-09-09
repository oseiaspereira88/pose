---
slug: pose-doctor-reports-unread-policy-keys
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-adoption-stamp-stays-readable
priority: 1
components: pose-mcp
delivers:
---

# Spec: Say when a review policy key is not read

## 1. Intent

### Goal
Report a key in the review policy that the engine does not model, recovering
what dropping `DisallowUnknownFields` gave up.

### Business value
`pose-adoption-stamp-stays-readable` made the review policy decoder ignore keys
it does not know, because refusing the whole policy over one makes every future
field break every older binary reading the same repository — which happened with
`contract_adoptions`, and left the adopting instance unreadable to the engine it
was adopting from.

That spec recorded the cost in its own risks: a misspelled key is now silently
ignored rather than reported. `contract_adoption` instead of
`contract_adoptions`, `allow_criteria_reuse` instead of
`allow_criterion_reuse` — the file reads as configured and is not, and the
setting the operator believed they made does nothing.

Both near-misses above are one character from a real key. That is the case worth
catching: an obviously wrong key gets noticed, a nearly-right one does not.

### Constraints
- A finding, never a refusal. The refusal is what was removed on purpose.
- The known-key list is derived from the struct. A list restated beside it
  drifts the first time a field is added, and then reports a real key as
  unknown — a failure indistinguishable from the misspelling it exists to catch.

### Non-goals
- Suggesting the key the operator meant. Edit distance would guess, and a wrong
  guess in a governance diagnostic is worse than none; the hint lists what is
  read instead.

---

## 2. Requirements

### Functional
- R1: `pose doctor` shall report any top-level key in `.pose/policy/review.json`
  that the engine does not model, naming each.
- R2: The hint shall list the keys the engine does read.
- R3: A policy with only known keys shall report `ok`.
- R4: The known-key list shall be derived from `ReviewPolicy`, not restated.

### Non-functional
- Nothing about the decoder changes; the policy still loads with unknown keys.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go` — the derived key list
- `pose-mcp/internal/cli/doctor.go` — the diagnostic

### Artifacts
- created: .pose/specs/2026-09-09-pose-doctor-reports-unread-policy-keys.md
- created: .pose/changelogs/unreleased/pose-doctor-reports-unread-policy-keys.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_fixture_audit_test.go

### Technical risks
- The check reads the raw document, so a key nested inside a known object is not
  inspected. Only the top level is compared, which is where the fields the
  engine models live.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Derive the key list from the struct (R4)
- [x] Increment 2: Report unknown keys and list what is read (R1, R2, R3)

### Validation
- [x] Two near-miss keys reported, a clean policy accepted, and the derivation asserted

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: where the list of known keys comes from.
- Options considered: (a) a literal list beside the check; (b) reflection over
  the `ReviewPolicy` struct tags.
- Decision: (b).
- Rationale: (a) is simpler to read and wrong the first time someone adds a
  field — the check would then report a real key as unknown, which is the same
  message a misspelling produces and therefore untrustworthy exactly when it
  matters. (b) cannot drift, and the test asserts the derivation finds real
  fields and excludes the nested structs that follow it in the file.

---

## 6. Validation

### Strategy
Assert two near-miss keys are named, that a clean policy is accepted, and that
the key list is derived rather than restated.

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
- Notes: all eight packages pass. The test uses `contract_adoption` and
  `allow_criteria_reuse`, each one character from a real key, because an
  obviously wrong key gets noticed by a reader and a nearly-right one does not.
  A third test asserts the derivation returns real policy fields and excludes
  `id`, `disposition` and `rationale`, which belong to the criterion struct that
  follows `ReviewPolicy` in the same file — the exact keys a hand-written list
  would plausibly have picked up.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <doctor's review.policy-keys compares the raw document's top-level keys against ReviewPolicyKnownKeys; the test asserts both misspellings are named>
- R2 [satisfied] <the hint joins the derived list; asserted by checking it contains contract_adoptions>
- R3 [satisfied] <the same test rewrites the policy with only known keys and asserts ok>
- R4 [satisfied] <ReviewPolicyKnownKeys reflects over the struct tags; TestKnownPolicyKeysComeFromTheStruct asserts it finds real fields and excludes nested-struct ones>

### Known gaps
- Only top-level keys are compared. A misspelling inside `profiles` or
  `reviewer_independence` is not caught.
- The same silent-default exposure exists for every other policy file the engine
  reads; this covers the review policy alone.

---

## 7. Final Report

### Follow-ups

- [done] Extend the unread-key finding to the delivery, artifact and capability policies, which have the same silent-default exposure. Done in `pose-policy-keys-and-release-surface-coverage`, and to docs, release and state as well — seven policies, keys derived from the structs, with a check that fails if a shipped policy is neither held to its keys nor exempted in writing. — owner:unowned crit:medium review:2026-12-09
