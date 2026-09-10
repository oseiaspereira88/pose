---
slug: pose-emittable-analysis-evidence-classes
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-one-evidence-class-vocabulary
priority: 0
components: pose-mcp
delivers:
---

# Spec: Analysis results have evidence classes of their own

## 1. Intent

### Goal
Make `lint`, `typecheck`, `security-scan` and `contract` classes a check may
emit and a profile may demand, instead of results reported as `build`.

### Business value
`pose-one-evidence-class-vocabulary` closed the gap between what a profile could
demand and what a check could emit: two lists of sixteen and nine that agreed on
six, so ten classes were demandable and impossible.

It did not close the gap one level down. Static analysis, type checking,
dependency scanning and contract verification have no class, so they are
reported as `build` or declare nothing. A criterion asking for security
assurance is then satisfied by a successful compilation — a gate that reads as
met by evidence that says nothing about it, which is the same defect the
unification exists to prevent.

`node/lint` and `node/typecheck` declare no class at all today, so their results
are discarded when a review collects evidence: the checks run, pass, and
contribute nothing.

### Constraints
- Widening a closed set is an adoption cost. An instance whose profile demands
  one of the four cannot be read by an engine predating them.
- A class nobody can emit is the defect the parent spec closed. Adding a class
  must not recreate it.

### Non-goals
- Declaring classes for the fourteen checks that still have none. Whether `npm
  test` is `unit` or `integration` is a different question, with its own
  standing finding.
- Shipping a default check that emits `security-scan`.

---

## 2. Requirements

### Functional
- R1: `lint`, `typecheck`, `security-scan` and `contract` shall be classes a
  check may emit and a profile may demand.
- R2: The shipped stack checks that are exactly a lint or a typecheck shall
  declare that class.
- R3: Every class shall keep a producer in a stack that had one: moving `go vet`
  to `lint` shall not leave the Go stack unable to emit `build`.
- R4: The shipped stacks catalog and this repository's own shall be held equal,
  so a class declared in one reaches the other.

### Non-functional
- `pose validate` accepts a check declaring any of the four.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/delivery_surface.go` — the vocabulary
- `.pose/indexes/validation-matrix.json` — this repository's stacks
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go` — the shipped stacks

### Artifacts
- created: .pose/specs/2026-09-09-pose-emittable-analysis-evidence-classes.md
- renamed: .pose/changelogs/unreleased/pose-emittable-analysis-evidence-classes.md -> .pose/changelogs/v4.0.0/pose-emittable-analysis-evidence-classes.md
- created: pose-mcp/internal/pose/evidence_class_vocabulary_test.go
- created: pose-mcp/internal/scaffold/distpolicy/stacks_agreement_test.go
- modified: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/dist/.pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-09-08-pose-one-evidence-class-vocabulary.md

### Technical risks
- Moving `go vet` off `build` removes the only Go producer of that class. A
  `go build ./...` check restores it; without that step the change would have
  made a `build` criterion unsatisfiable for every Go module, and nothing in the
  engine would have reported it — `review.evidence-vocabulary` checks the global
  vocabulary, not per-stack producers.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The four classes join the vocabulary (R1)
- [x] Increment 2: lint and typecheck declared where they apply (R2)
- [x] Increment 3: A Go build check, so `build` keeps a producer (R3)
- [x] Increment 4: The two stacks catalogs held equal (R4)

### Validation
- [x] The catalog agreement shown failing when one side is changed alone

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: whether to add all four classes, only `security-scan`, or none.
- Decision: all four. Recorded as an amendment to the delivery-integrity ADR.
- Rationale: `security-scan` is the case whose consequence leaves the
  repository, but `lint` and `typecheck` have shipped checks today that declare
  nothing, so adding only `security-scan` would leave the two the vocabulary
  could already have covered still discarded.

### Decision 2
- Date: 2026-09-09
- Context: `security-scan` and `contract` have no shipped default check, which
  looks like the defect the parent spec closed — a class nobody emits.
- Decision: add them anyway.
- Rationale: it is a different situation. `validation` was refused by `pose
  validate`, so no check anywhere could emit it. These are accepted, so a
  project registers its own; the shipped catalog is a default, not the universe
  of checks. A shipped default that emits `security-scan` is a follow-up.

### Decision 3
- Date: 2026-09-09
- Context: the stacks catalog exists twice — in this repository's matrix and as
  a Go literal in `distpolicy.go`, because the file is excluded from the
  byte-for-byte sync over `moduleOverrides` (issue #22).
- Decision: hold the two equal with a test.
- Rationale: they agreed, by hand, with nothing checking. The first edit of this
  spec changed one and not the other, and it would have shipped a default
  catalog disagreeing with the one this repository validates itself against.

---

## 6. Validation

### Strategy
Assert the vocabulary, and assert the two catalogs agree — the second shown
failing when one side is edited alone.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose validate --tolerant --module pose-mcp`
- Scope: this instance
- Expected: SUCCESS

### Execution log
- Date: 2026-09-09
- Environment: local, Go 1.26
- Notes: `pose doctor` goes from 16 classless registered checks to 14, the two
  being `node/lint` and `node/typecheck`. `pose validate --tolerant --module
  pose-mcp` reports SUCCESS with the added `go build ./...` check. Reverting the
  `go vet` class in `distpolicy.go` alone fails the catalog agreement test with
  the message naming both files.

  The adoption cost was observed rather than argued: running `pose check
  --strict` with a binary built before this change fails with `stacks.node check
  lint has unknown evidenceClass "lint"`, repeated once per governed spec. That
  is the closed set doing its job, and it is exactly what an engine predating
  these four classes does with a matrix that declares them.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <ValidEvidenceClasses carries the four; TestAnalysisClassesAreEmittable fails if any is absent, and TestTheVocabularyStaysOneSet holds the operator-facing list to the accepted set>
- R2 [satisfied] <node/lint declares lint, node/typecheck declares typecheck, and go/vet moved from build to lint, in both catalogs>
- R3 [satisfied] <a `go build ./...` check declaring build was added to the Go stack; `pose validate --tolerant --module pose-mcp` runs it and reports SUCCESS>
- R4 [satisfied] <TestShippedStacksCatalogMatchesThisRepositorys compares `stacks` and `deliveryProfiles` between distpolicy.go's literal and .pose/indexes/validation-matrix.json, and was shown failing on a one-sided edit>

### Known gaps
- Fourteen shipped checks still declare no class, so their results are discarded
  when a review collects evidence.
- No shipped check emits `security-scan` or `contract`.
- The refusal above is per-repository: an instance that declares one of the four
  in its matrix cannot be validated by an older engine at all, not only when a
  profile demands the class. No shipped profile demands one, so a fresh instance
  takes no cost until it opts in — but declaring the class on a check is enough.
- Nothing reports that a stack has no producer for a class a profile demands;
  the vocabulary check is global, not per-stack. That is how moving `go vet`
  would have gone unnoticed.

---

## 7. Final Report

### Summary
Analysis results are no longer reported as builds, and the two stacks catalogs
can no longer disagree.

### Follow-ups

- [open] Ship a default check that emits `security-scan`, so the class has a producer out of the box rather than only when a project registers one — owner:unowned crit:low review:2026-12-09
- [open] Report when a stack has no check emitting a class its selected profiles demand; the vocabulary check is global, so moving `go vet` off `build` left the Go stack with no `build` producer and nothing said so — owner:unowned crit:medium review:2026-12-09
