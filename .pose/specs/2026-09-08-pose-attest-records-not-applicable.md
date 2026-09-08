---
slug: pose-attest-records-not-applicable
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-attestation-evidence-must-be-in-the-bundle
priority: 1
components: pose-mcp
delivers:
---

# Spec: A reviewer can record a criterion as not applicable

## 1. Intent

### Goal
Let `pose review attest` record the dispositions the engine already models, so a
criterion that genuinely does not apply has an honest expression.

### Business value
The engine has modelled three criterion dispositions since sealed bundles were
introduced — `passed`, `not-applicable` and `finding` — and
`validateBundleAttestation` requires a rationale for the second. `pose review
attest` could write only the first: every required criterion in the plan became
`passed`, with evidence picked from the refs the reviewer supplied.

That was harmless while nothing checked the evidence. Once a `passed` must cite
sealed evidence of a demanded class, it stops being harmless and becomes a dead
end. An adopting repository hit it immediately: a spec that fixed a port
collision in a test harness carries a `frontend-accessibility` criterion
requiring `a11y`, and that instance registers no check emitting `a11y`. The
reviewer's options were to claim the criterion passed on a `build` result that
does not support it, or to leave the closeout unable to complete. The change
has no user-visible surface; "not applicable, and here is why" is the true
statement, and it could not be written.

`pose review auto-attest` gained exactly that ability in
`pose-attestation-evidence-must-be-in-the-bundle`: where a scope carries no
delivery target it records the criterion `not-applicable` with a rationale
naming what is missing. A human reviewer should not have fewer options than the
automated path.

### Constraints
- The default does not change. A criterion nobody names is still `passed` with
  evidence picked from the supplied refs, so every existing invocation behaves
  as before.
- A `not-applicable` without a rationale is refused at the flag, not only at
  sealing. The reviewer can act on the flag they typed.

### Non-goals
- Letting `--criterion` reach an optional criterion or one outside the plan.
  An attestation speaks about the plan it approves.
- A disposition for tools. `--tool` already carries one.

---

## 2. Requirements

### Functional
- R1: `--criterion ID|disposition|evidence|rationale` shall set that criterion's
  disposition in the attestation.
- R2: The accepted dispositions shall be `passed`, `not-applicable` and
  `finding`, matching what the engine validates.
- R3: `not-applicable` without a rationale shall be refused, naming the flag.
- R4: A criterion not named shall keep the current default.
- R5: A `--criterion` naming something outside the plan's required criteria, or
  naming one twice, shall be refused.
- R6: A criterion dispositioned `finding` shall name a finding this attestation
  records.

### Non-functional
- The existing review suite passes unchanged.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/review_closeout.go` — criterion dispositions
- `pose-mcp/internal/cli/help_catalog.go` — the command reference

### Artifacts
- created: .pose/specs/2026-09-08-pose-attest-records-not-applicable.md
- created: .pose/changelogs/unreleased/pose-attest-records-not-applicable.md
- created: pose-mcp/internal/cli/review_criterion_disposition_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go

### Technical risks
- `not-applicable` is a way past a criterion, and this makes it reachable by
  hand. It is bounded by requiring a rationale that is recorded in the immutable
  attestation, so the reason is as durable and as reviewable as the decision —
  which is the same guarantee the engine already relies on for the automated
  path.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Parse and apply --criterion, keeping the default (R1, R4)
- [x] Increment 2: Refuse what the attestation could not stand behind (R2, R3, R5)
- [x] Increment 3: Tie a finding criterion to a recorded finding (R6)

### Validation
- [x] Each rejection asserted, and the override shown to be load-bearing

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: the pipe-delimited shape.
- Decision: `ID|disposition|evidence|rationale`, mirroring `--tool`'s
  `ID|component|disposition|evidence|rationale`.
- Rationale: the command already has one multi-field flag and a reviewer using
  both should not have to remember two conventions. The rationale stays last, so
  the common `ID|not-applicable||<why>` reads the same way as the tool form.

### Decision 3
- Date: 2026-09-08
- Context: review pointed out that `--criterion ID|finding||` is accepted with no
  finding filed, and that an `approved` attestation then passes verification —
  `validateBundleAttestation` checks the disposition and afterwards only rejects
  findings that exist and remain open. So a criterion the reviewer explicitly did
  not pass reported a clean closeout.
- Decision: a `finding` criterion must name a finding this attestation records,
  in the evidence slot.
- Rationale: the hole predates this change, but `--criterion` is what made it
  reachable by hand, so filing it as a follow-up would have meant shipping the
  reachability and deferring the guard. The evidence slot already exists and a
  finding id is exactly the reference the disposition is claiming, so this costs
  no new syntax.

### Decision 2
- Date: 2026-09-08
- Context: whether to refuse a missing rationale here, given sealing refuses it
  anyway.
- Decision: refuse at the flag.
- Rationale: the later refusal names a criterion inside an artifact the reviewer
  has not written yet; this one names the flag they just typed and shows the
  shape to use. Both remain — the engine's check is what makes the guarantee,
  and this is what makes it actionable.

---

## 6. Validation

### Strategy
Assert the override, assert that everything unnamed is untouched, and assert
each refusal separately.

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
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass. Disabling the override makes the test report
  `{ID:frontend-accessibility Disposition:passed Evidence:unit:api/go/test
  Rationale:}` — which is precisely the state this spec exists to correct: a
  criterion claiming to have passed on evidence of a class it never asked for.
  Eight refusal cases are asserted individually.

### Results summary
- Successes: R1, R2, R3, R4, R5, R6 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <reviewCriterionDispositions applies the override; TestAttestRecordsNotApplicableWithARationale asserts the disposition, the rationale and the absent evidence>
- R2 [satisfied] <the disposition switch accepts exactly the three the engine validates; the "unknown disposition" case asserts the refusal>
- R3 [satisfied] <the "no rationale" case asserts the error names the flag shape>
- R4 [satisfied] <the same test asserts the unnamed criterion keeps passed with the picked evidence, and that optional criteria stay out>
- R5 [satisfied] <the "criterion not in the plan", "optional criterion" and "duplicate" cases>
- R6 [satisfied] <the finding branch requires the evidence slot to name a finding present in the parsed --finding list; TestAttestTiesAFindingCriterionToARecordedFinding asserts the accepted form and that the same criterion is refused with no finding recorded, plus two refusal cases for an empty and an unknown id>

### Known gaps
- The finding is named by id, and nothing checks that it is the finding that
  actually describes this criterion's problem. That is a judgement the record
  carries rather than a property the engine can verify.

---

## 7. Final Report

### Follow-ups

- [open] Check that a finding named by a criterion is the one describing that criterion's problem, not merely a recorded id — owner:unowned crit:low review:2027-01-08
