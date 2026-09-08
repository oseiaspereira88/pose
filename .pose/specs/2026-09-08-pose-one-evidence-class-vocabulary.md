---
slug: pose-one-evidence-class-vocabulary
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-tool-dispositions-must-be-supported
priority: 0
components: pose-mcp
delivers:
---

# Spec: One vocabulary of evidence classes

## 1. Intent

### Goal
Make what a review profile may demand and what a registered check may emit the
same list, so a profile cannot express a gate nothing can pass.

### Business value
Two lists governed evidence classes and disagreed in both directions:

- `reviewEvidenceClassCatalog` — sixteen names a profile could declare.
- `ValidEvidenceClasses` — nine a check may emit, enforced by `pose validate`.

They agreed on six. **Ten** names a profile could demand were impossible to
satisfy — `contract`, `integration-test`, `lint`, `manual-review`,
`observability`, `requirement-trace`, `security-scan`, `test`, `typecheck`,
`validation` — and **three** a check could emit could never be demanded:
`contrast`, `design-system`, `visual-regression`.

A profile demanding an unproducible class plans a gate that only a fabricated
disposition can pass. That is the failure four specs have closed downstream, one
consumer at a time: the tool class filter in 1.7.12, the plan-time criterion
blocker and the profile reconciliation in
`pose-attestation-evidence-must-be-in-the-bundle`, the attestation checks in the
same, and the tool half in `pose-tool-dispositions-must-be-supported`. Each was
correct and none reached the cause, because the cause is that the contract
admits the input at all.

Refusing the profile is where it stops being expressible — and it makes the
downstream guards unnecessary rather than merely redundant.

### Constraints
- The vocabulary is the one a check may emit. Widening it instead would keep
  every profile working, but it would ratify names no check produces and let a
  check declare `manual-review` as its evidence class.
- The message must say what may be written, not only what may not.

### Non-goals
- Adding classes. Four of the ten dropped names — `lint`, `typecheck`,
  `security-scan`, `contract` — describe real, distinct, checkable things that
  today are classified `build`. Making them emittable is a defensible change and
  a different one; doing it inside a unification would make it impossible to
  tell which break came from which, and each needs a check that actually emits
  it to mean anything. Recorded as a follow-up.

---

## 2. Requirements

### Functional
- R1: A review profile declaring an evidence class outside `ValidEvidenceClasses`
  shall fail to load, in criteria and in tools alike.
- R2: The error shall name the offending class and list the vocabulary.
- R3: Every class a check may emit shall be demandable by a profile.
- R4: The downstream filters this makes unreachable shall be removed, not left
  as branches nothing can enter.

### Non-functional
- The shipped profiles load unchanged; they were reconciled to the intersection
  in `pose-attestation-evidence-must-be-in-the-bundle`.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go` — profile contract validation
- `pose-mcp/internal/pose/review_plan.go` — the filters this removes
- `pose-mcp/internal/pose/delivery_surface.go` — the vocabulary and its listing

### Artifacts
- created: .pose/specs/2026-09-08-pose-one-evidence-class-vocabulary.md
- renamed: .pose/changelogs/unreleased/pose-one-evidence-class-vocabulary.md -> .pose/changelogs/v2.0.0/pose-one-evidence-class-vocabulary.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_plan_test.go
- modified: pose-mcp/internal/pose/delivery_surface.go

### Technical risks
- An instance whose own profile declares any of the ten dropped names stops
  loading it, which fails review planning rather than degrading it. That is the
  intended severity — the alternative is the plan it was building — but it is a
  hard stop on update, and `pose doctor`'s `review.evidence-vocabulary` has
  reported exactly this since 1.8.0 for anyone who ran it.
- Three classes become demandable that never were. Nothing depends on them being
  refused, but a profile can now ask for evidence an instance may have no check
  for, which is a different failure and one the instance can fix.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Validate profile classes against the single vocabulary (R1, R2, R3)
- [x] Increment 2: Remove the filters it makes unreachable (R4)

### Validation
- [x] Both directions asserted, and the refusal shown to be load-bearing

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: which list survives.
- Options considered: (a) the profile catalogue, widening what a check may emit;
  (b) `ValidEvidenceClasses`, narrowing what a profile may demand.
- Decision: (b).
- Rationale: (a) keeps every existing profile working and is the tempting
  answer, but it ratifies a vocabulary where `test` sits beside `unit`,
  `integration` and `e2e` — the overlap an earlier reconciliation was undoing —
  and it would let `pose validate` accept `manual-review` as a check's evidence
  class, which is not a thing a check can produce. What a check may emit is the
  only one of the two lists grounded in something that runs.

### Decision 2
- Date: 2026-09-08
- Context: what to do with the filters that became unreachable — the criterion
  blocker and the tool class drop, both added in the last two weeks.
- Decision: remove them.
- Rationale: with the profile refused at load, neither can be entered by any
  input the contract admits: profile criteria and profile tools both pass
  through `validateReviewContractRefs`, and the one synthetic criterion is added
  afterwards with a hard-coded `integration`. An unreachable branch with no test
  is dead code that reads as a live guarantee. Removing them is also the
  clearest statement of what this change bought: two guards became unnecessary,
  not merely redundant.

### Decision 3
- Date: 2026-09-08
- Context: whether to also make `lint`, `typecheck`, `security-scan` and
  `contract` emittable, since they name real check kinds that today report as
  `build`.
- Decision: not here.
- Rationale: it is a defensible change and a separate one. Bundling it would mix
  a narrowing and a widening in the same release, so an adopting instance that
  breaks could not tell which half broke it, and each of the four is only worth
  having once a check emits it — which is work in the instance, not in the
  engine.

---

## 6. Validation

### Strategy
Assert both directions of the disagreement — the class a profile may no longer
demand, and the classes it now may — and confirm the refusal is what produces
the first.

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
- Notes: all eight packages pass, including every review test with both filters
  removed — which is the evidence that nothing depended on them. Disabling the
  refusal fails the load test on "a profile demanding a class no check may emit
  was accepted". The shipped profiles load untouched, because they were
  reconciled to the intersection two specs ago; had they not been, this change
  would have broken POSE's own instance first.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <validateReviewContractRefs checks ValidEvidenceClasses and runs on both criteria and tools at profile load; TestProfileDeclaringAClassNoCheckMayEmitDoesNotLoad asserts the refusal>
- R2 [satisfied] <the error names the class and joins the sorted vocabulary; the test asserts the class, the phrase and two vocabulary members appear>
- R3 [satisfied] <TestProfileMayDemandEveryClassACheckMayEmit loads a profile for each of design-system, contrast and visual-regression, which the old catalogue refused>
- R4 [satisfied] <composeReviewCriteria's blocker, buildReviewTools' drop and producibleEvidenceClasses are gone; the suite passes without them>

### Known gaps
- The vocabulary is a compiled-in list. A project that runs a genuinely
  different kind of check has no way to name it, and the answer today is to
  classify it as the nearest existing class — which is how `lint`, `typecheck`
  and `security-scan` results all report as `build`.

---

## 7. Final Report

### Follow-ups

- [open] Decide whether lint, typecheck, security-scan and contract should be emittable classes rather than reported as build — owner:unowned crit:medium review:2026-12-08
- [open] Consider whether the evidence-class vocabulary should be extensible by an instance rather than compiled in — owner:unowned crit:low review:2027-03-08
