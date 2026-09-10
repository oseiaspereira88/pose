---
slug: pose-class-producers-reads-the-disjunction
status: in-progress
completed_at:
created_at: 2026-09-10
supersedes:
depends_on: pose-report-a-demanded-class-nothing-produces
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: The class-producers check reports a criterion, not a class

## 1. Intent

### Goal
Report a review criterion none of whose accepted evidence classes is produced,
and skip a profile that cannot be selected here at all.

### Business value
`validate.class-producers` shipped in v5.0.0 reporting two classes in this
repository, `a11y` and `e2e`. Acting on that report showed both were wrong in
different ways, and it took acting on it to see either.

`evidence_classes` on a criterion is a disjunction: the attestation cites one of
them and the first match satisfies it. The check reported each class on its own,
so `e2e` was named while every criterion listing it also accepted `integration`
or `unit`, both of which are produced. A true statement that named nothing to
fix — which is how a check earns being ignored, and the failure mode the spec
that introduced it was explicitly trying to avoid.

`a11y` was named because `frontend-review` demands it. That profile is selected
by `languages: [javascript, typescript]`, and this repository has two Go modules
and a Python docs site. The criterion can never apply here, so it is not one
anyone will be held to.

Neither correction weakens the check. The case it was built for still fires: a
criterion whose only accepted class lost its producer.

### Constraints
- Evaluating selectors fully needs the component-aware planner. A partial
  evaluation must only ever make the check quieter, never wrongly loud.

### Non-goals
- Registering an `a11y` check here. The follow-up offered that or reconciling
  the profile; the measurement says neither is needed, because the criterion
  never applies.

---

## 2. Requirements

### Functional
- R1: A criterion shall be reported only when none of the classes it accepts is
  produced.
- R2: A criterion accepting at least one produced class shall not be reported.
- R3: A profile whose language selector names no domain present in the
  repository shall be skipped.
- R4: A component in that language appearing shall bring the profile back.

### Non-functional
- Only the language selector is evaluated, and only to skip.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go` — the `validate.class-producers` check

### Artifacts
- created: .pose/specs/2026-09-10-pose-class-producers-reads-the-disjunction.md
- created: .pose/changelogs/unreleased/pose-class-producers-reads-the-disjunction.md
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_class_producers_test.go
- modified: .pose/specs/2026-09-10-pose-report-a-demanded-class-nothing-produces.md

### Technical risks
- Skipping by language selector could hide a criterion that would apply through
  another selector. It only skips when the profile names languages and none is
  present, so a profile selected by component id or delivery kind is unaffected.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Report the criterion, reading its classes as a disjunction (R1, R2)
- [x] Increment 2: Skip a profile no component can select (R3, R4)

### Validation
- [x] The case the check exists for still fires

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the follow-up asked to register `a11y` and `e2e` checks here, or
  reconcile the profiles demanding them.
- Decision: neither — fix the check instead.
- Rationale: `e2e` was never unsatisfiable, and `a11y` belongs to a profile no
  component here can select. Registering an accessibility check for a Go CLI and
  a docs site would be inventing a surface to satisfy a criterion that never
  applies, and removing the overlay would lose it for the day a JavaScript
  component appears. The report was wrong, not the repository.

### Decision 2
- Date: 2026-09-10
- Context: this is the second correction to a check that is two days old.
- Decision: record it as such rather than folding it into the original spec.
- Rationale: both defects were found by acting on the check's output, not by
  reviewing it — the first because a follow-up said to act, the second while
  deciding how. That is worth being able to read later, and rewriting the
  original spec would hide it.

---

## 6. Validation

### Strategy
Assert both corrections and require the original case to keep failing.

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
- Notes: before, this repository reported `a11y, e2e`. Reading the disjunction
  reduced it to one criterion, `frontend-review/frontend-accessibility (a11y)`;
  skipping the unselectable profile cleared it. The motivating case still fires:
  a matrix whose only check emits `unit` reports the criterion demanding `e2e`,
  and moving that check to `lint` reports the `unit` one as well.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <the check walks criteria and tools, and reports one only when none of its classes is produced; TestDoctorReportsADemandedClassNoCheckProduces requires the criterion id and its class set in the message>
- R2 [satisfied] <TestACriterionAcceptingAProducedClassIsNotReported gives a criterion `e2e|unit` with only `unit` produced and requires silence>
- R3 [satisfied] <TestAProfileForAnAbsentLanguageIsNotReported has an overlay selected by typescript in a repository with only a Go module, and requires ok>
- R4 [satisfied] <the same test then adds a typescript module and requires the criterion to be reported>

### Known gaps
- Only the language selector is evaluated. A profile selected by component id,
  delivery kind or criticality is always considered, so a criterion that cannot
  apply for one of those reasons is still reported.

---

## 7. Final Report

### Summary
The check names a criterion someone will actually be held to, instead of a class
that happens to have no producer.

### Follow-ups

- [open] Resolve profile selection through the component-aware planner rather than by reading the language selector, so a profile that cannot apply for any reason is skipped and not only for an absent language — owner:unowned crit:low review:2027-03-10
