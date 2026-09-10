---
slug: pose-report-coverage-that-rests-on-inference
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-component-evidence-is-not-inherited-upward
priority: 0
components: pose-mcp
delivers:
task_type: feature
---

# Spec: Coverage that rests on an inference is visible

## 1. Intent

### Goal
Report a delivery target whose only passing evidence for a class comes from a
module containing it, rather than from a result naming the target's own module.

### Business value
`pose-component-evidence-is-not-inherited-upward` settled the direction:
evidence answers downward, because a module-wide run does exercise its subtree.
Usually. A run configured to skip a directory inside it does not, and nothing in
the result says which. That is the one part of the containment rule POSE cannot
check, and it was recorded there as a known gap.

Closing it properly means a check declaring what it walked — a change to the
validation result contract and a migration for every project. Designing that
contract now would mean choosing its shape against no observed case: measured
here, this repository's targets and results name the same module, so every match
is by equality or by the root and not one rests on the inference.

Reporting it costs nothing, refuses nothing, and produces exactly the evidence
that decision needs: how many targets, in real repositories, are gated on a run
of something larger than themselves.

### Constraints
- A warning. Refusing would break the layout where a module runs its checks
  once, which is the option the previous spec rejected on purpose.

### Non-goals
- Declaring the walked subtree. That is the eventual contract and it is deferred
  until there is a case to calibrate it against.

---

## 2. Requirements

### Functional
- R1: A target whose only passing evidence for a class comes from a containing
  module shall carry an `inferred-coverage` finding naming the class, the target
  module and the remedy.
- R2: A result naming the target's own module, or the repository root, shall not
  be reported.
- R3: One result naming the target's own module shall silence the report for
  that class, even when a containing module's result also matches.
- R4: The target shall keep its coverage.

### Non-functional
- The finding is emitted once per target, listing the classes, rather than once
  per result.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/delivery_surface.go` — the classifier and the finding

### Artifacts
- created: .pose/specs/2026-09-10-pose-report-coverage-that-rests-on-inference.md
- created: .pose/changelogs/unreleased/pose-report-coverage-that-rests-on-inference.md
- created: pose-mcp/internal/pose/inferred_coverage_test.go
- modified: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: .pose/specs/2026-09-09-pose-component-evidence-is-not-inherited-upward.md

### Technical risks
- A repository laid out with checks at the module root and targets inside it
  sees one finding per target and class. That is the intended visibility, and it
  is what makes the deferred decision measurable — but it is also noise until
  someone acts on it, so the severity is a warning and the remedy names both
  ways out.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Classify the match and report the inferred case (R1, R2, R4)
- [x] Increment 2: An exact result silences it (R3)

### Validation
- [x] Confirmed to change nothing in this repository

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: four ways to close the gap the previous spec left, put to the
  maintainer with the measurement.
- Decision: report the inference now; defer the declared-subtree contract.
- Rationale: the contract is the durable answer and this repository offers no
  case to calibrate it against — every match here is by equality or by the root.
  Designing a migration for every project against zero observed use is the shape
  of decision this cycle has repeatedly been better off measuring first.

### Decision 2
- Date: 2026-09-10
- Context: whether one inferred result should be reported when an exact one also
  exists.
- Decision: no.
- Rationale: the question an operator acts on is whether the target's coverage
  rests on the inference. A target with a check of its own does not, whatever
  else also matched.

---

## 6. Validation

### Strategy
Assert each way a match can happen, and confirm the repository's own graph is
unchanged.

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
- Notes: `surface-check` on this repository reports zero `inferred-coverage`
  findings, which is what the measurement predicted: its 134 targets name module
  `pose-mcp` or `.`, its results name `pose-mcp`, and all 695 matching pairs
  resolve by equality or by the root. The behaviour is carried by fixtures.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestCoverageFromAContainingModuleIsReported requires the finding to name the class and the target module and to carry a remedy>
- R2 [satisfied] <TestExactAndRootCoverageAreNotReported covers a result naming the target's own module and a root result>
- R3 [satisfied] <TestAnExactResultSilencesTheReport gives the target both a containing result and its own, and requires silence>
- R4 [satisfied] <the first test also requires no unconnected-artifact or surface-without-provenance finding, so the target is reported and still covered>

### Known gaps
- The finding cannot distinguish a module-wide run that genuinely covers the
  target from one that skips it. That distinction is what the deferred contract
  would provide, and this exists to measure whether it is worth building.

---

## 7. Final Report

### Summary
The one inference left in the containment rule is visible, and the contract that
would remove it can now be designed against real cases.

### Follow-ups

- [open] Revisit the declared-subtree contract once `inferred-coverage` has been observed in real repositories — the four approaches and why each was set aside are in the ADR amendment, and the missing input was how much anyone actually relies on the inference — owner:unowned crit:low review:2027-06-10
