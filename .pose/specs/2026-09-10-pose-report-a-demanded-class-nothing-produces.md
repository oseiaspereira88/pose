---
slug: pose-report-a-demanded-class-nothing-produces
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-emittable-analysis-evidence-classes
priority: 0
components: pose-mcp
delivers:
task_type: feature
---

# Spec: Report an evidence class the profiles demand and nothing produces

## 1. Intent

### Goal
Report, as a `pose doctor` finding, an evidence class the selected review
profiles demand that no registered check in this repository produces.

### Business value
`review.evidence-vocabulary` asks whether a class *could* be emitted by some
check somewhere: it holds a profile to the closed vocabulary, so a profile
cannot demand a class the engine would refuse to register. It says nothing about
whether this repository actually generates that evidence.

The two came apart in `pose-emittable-analysis-evidence-classes`. Moving `go
vet` from `build` to `lint` left no registered check producing `build`, while the
selected profiles kept demanding it. `build` was still a valid class, so the
vocabulary check stayed green. A criterion depending on evidence the repository
does not generate, with nothing saying so — noticed only because the same change
added a `go build ./...` check by hand, and recorded then as a known gap.

Run here, the new check has two findings immediately: the selected profiles
demand `a11y` and `e2e`, and no registered check emits either.

### Constraints
- A warning, not an error. Evidence can be imported from elsewhere, so a class
  with no local producer is a fact worth reporting rather than a refusal.

### Non-goals
- Deciding per module or per stack whether a class is reachable. Resolving which
  profile applies to which component is the planner's job, and the useful
  question here is simpler: does this repository generate the evidence at all.

---

## 2. Requirements

### Functional
- R1: A class the selected profiles demand that no stack check and no module
  override produces shall be reported, named, with the remedy.
- R2: A class a check does produce shall not be named.
- R3: A module override shall count as a producer.
- R4: An instance whose selected profiles demand no class shall produce no
  finding.

### Non-functional
- The check reads the instance's own matrix and profiles.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go` — the new `validate.class-producers` check,
  beside the vocabulary check whose blind spot it covers

### Artifacts
- created: .pose/specs/2026-09-10-pose-report-a-demanded-class-nothing-produces.md
- renamed: .pose/changelogs/unreleased/pose-report-a-demanded-class-nothing-produces.md -> .pose/changelogs/v5.0.0/pose-report-a-demanded-class-nothing-produces.md
- created: pose-mcp/internal/cli/doctor_class_producers_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: .pose/specs/2026-09-09-pose-emittable-analysis-evidence-classes.md

### Technical risks
- It reuses the selected-profile set the vocabulary check computes, so a change
  to how profiles are selected moves both together. That is the intent: they are
  two questions about the same set.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The check reports, accepts and stays quiet (R1, R2, R3, R4)

### Validation
- [x] The motivating case reproduced: the only producer moves to another class

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the follow-up asked for a per-stack report; resolving which profile
  applies to which stack is the component-aware planner's work.
- Decision: report per repository, not per stack.
- Rationale: the harm the follow-up describes — nothing produces `build` any
  more — is visible without the mapping, and the mapping would have to guess
  which profile governs which module. A per-stack version that guessed would
  report a Go module for not emitting `a11y`, which is noise, and teaching an
  operator to ignore this check would cost more than the precision buys.

---

## 6. Validation

### Strategy
Reproduce the case that motivated it — the only producer of a demanded class
starts producing something else — and require the finding to name it.

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
- Notes: on this repository the check reports `a11y, e2e` — two classes the
  selected profiles demand and nothing produces. The motivating case is
  reproduced in a fixture: a matrix whose only check emits `unit` reports `e2e`;
  moving that check to `lint` reports `unit` as well.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestDoctorReportsADemandedClassNoCheckProduces requires the warn, the class name and a hint>
- R2 [satisfied] <the same test requires the produced class not to be named — naming it would tell an operator to register a check that already exists>
- R3 [satisfied] <TestAModuleOverrideCountsAsAProducer requires an override's class to count>
- R4 [satisfied] <TestNoFindingWhenNothingIsDemanded requires no finding at all when the profiles demand no class>

### Known gaps
- The check cannot tell a class that is imported from elsewhere on purpose from
  one nobody thought about. Both read as the same warning.
- `a11y` and `e2e` are reported here and both are real: this repository demands
  them and generates neither.

---

## 7. Final Report

### Summary
A criterion demanding evidence this repository does not generate is visible,
rather than waiting for a closeout to fail on it.

### Follow-ups

- [open] Let a project record that a demanded class is imported on purpose, so the finding distinguishes evidence that arrives from elsewhere from evidence nobody produces — owner:unowned crit:low review:2027-03-10
- [open] Register checks emitting `a11y` and `e2e` in this repository, or reconcile the profiles that demand them — the new check reports both, and both are real — owner:@pose-maintainers crit:medium review:2026-12-10
