---
slug: pose-roadmap-check-reaches-its-gate
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-remaining-command-surfaces-coverage
priority: 0
components: pose-mcp
delivers:
---

# Spec: roadmap-check evaluates the criteria it was asked about

## 1. Intent

### Goal
Stop the delivery graph builder from returning before it loads a roadmap, so
`pose roadmap-check` answers about the cut criteria a roadmap declares instead
of about an empty list.

### Business value
`extendCurrentDeliveryGraph` returned early when `.pose/indexes/validation-matrix.json`
or `.pose/specs` was absent — before a single roadmap was read. `roadmap-check`
then reported zero cut criteria and exited 0 on a repository whose roadmap
declared several.

A gate that cannot evaluate its criteria was reading as a gate that passed them,
in both `--strict` and `--tolerant`. It is the same shape this cycle keeps
finding: the path that would fail is never reached, so the absence of a failure
is read as a pass.

Neither absence justifies skipping the evaluation. Without profiles there are no
delivery targets, so a criterion naming one is unresolved — which the criteria
loop already reports as `unknown delivery ref`, and which is the honest answer.
A `check:` or `manual-review:` ref needs no profile index at all, and those were
being skipped for a precondition they do not have.

It was found while writing a coverage fixture, not by a report: the fixture had
no profile index, and both modes returned 0 on a criterion naming a delivery
target that does not exist.

### Constraints
- A repository that has adopted delivery integrity must be unaffected.

### Non-goals
- Changing what an unresolved criterion means. `unknown delivery ref` is already
  the right verdict; the defect was never reaching it.

---

## 2. Requirements

### Functional
- R1: An absent profile index shall not skip roadmap evaluation; the criteria
  shall be read and judged with no delivery targets available.
- R2: An absent specs directory shall likewise not skip it.
- R3: A criterion needing no delivery profile — `manual-review:` — shall be
  judged on its own terms.
- R4: A criterion the gate can satisfy shall still pass, so a repository without
  delivery profiles is not newly blocked.

### Non-functional
- A repository with delivery profiles produces the same graph as before.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/surface_check.go` — the two early returns

### Artifacts
- created: .pose/specs/2026-09-09-pose-roadmap-check-reaches-its-gate.md
- renamed: .pose/changelogs/unreleased/pose-roadmap-check-reaches-its-gate.md -> .pose/changelogs/v4.0.0/pose-roadmap-check-reaches-its-gate.md
- created: pose-mcp/internal/cli/roadmap_check_gate_test.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: .pose/specs/2026-09-09-pose-remaining-command-surfaces-coverage.md

### Technical risks
- Continuing past the profiles branch reaches `ListSpecs` in a case that used to
  return first, and the store wraps its error — `os.IsNotExist` does not unwrap,
  so the wrapped absence became a hard failure of `pose index`. Both checks use
  `errors.Is(err, fs.ErrNotExist)` now. The suite caught it immediately, which
  is the only reason it is a note rather than a defect.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Neither absence skips the roadmap evaluation (R1, R2, R3, R4)

### Validation
- [x] The tests shown failing against the early return

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: two ways to make the gate honest — evaluate the criteria anyway, or
  have `roadmap-check` refuse to answer when it could not evaluate.
- Decision: evaluate them.
- Rationale: refusing would be right for delivery refs and wrong for the rest —
  a `manual-review:` criterion is fully judgeable without a profile index, and
  telling its author that the gate cannot run would be false. Evaluating gives
  every criterion the verdict it deserves, and an unresolved delivery ref
  already has one.

### Decision 2
- Date: 2026-09-09
- Context: continuing past the early return changes what every consumer of the
  graph sees, not only `roadmap-check`.
- Decision: accept it, having measured.
- Rationale: `surface-check` on this repository reports the same 524
  `validated-by` edges and the same 371 findings before and after, because it
  has a profile index and never took the early return. The repositories affected
  are those with roadmaps and no profile index — where the previous answer was
  wrong.

---

## 6. Validation

### Strategy
Run the gate on a repository with a roadmap and nothing else, and require it to
judge the criteria; require the tests to fail when the early return is restored.

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
- Notes: restoring the early return fails both evaluation tests. `surface-check`
  on this repository reports 524 `validated-by` edges and 371 findings before
  and after. The first run after the change failed
  `TestIndexUsesConfiguredMetadataDefaults`, because the newly reached
  `ListSpecs` returns a wrapped not-exist that `os.IsNotExist` does not unwrap.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestRoadmapCheckEvaluatesCriteriaWithoutDeliveryProfiles runs the gate on a repository with no profile index and requires the criterion to be judged, unpassed, with a blocker and a non-zero exit>
- R2 [satisfied] <the same branch treats an absent specs directory as empty rather than as a reason to return, using errors.Is so the store's wrapped error is recognised>
- R3 [satisfied] <TestRoadmapCheckEvaluatesAManualReviewCriterion requires a manual-review ref pointing outside the project to block, with no profile index present>
- R4 [satisfied] <TestRoadmapCheckStillPassesACriterionItCanSatisfy requires a confined manual-review ref whose report exists to pass>

### Known gaps
- `roadmap-check` still answers 0 for a roadmap that declares no criteria at
  all. That is correct, and indistinguishable at the exit code from a roadmap
  whose criteria could not be read — which no longer happens, but nothing
  asserts the distinction.

---

## 7. Final Report

### Summary
The gate reads the criteria it was asked about, whether or not the repository
has adopted delivery integrity.

### Follow-ups

- [open] Have `roadmap-check` say when a roadmap declares no cut criteria, so a roadmap with nothing to gate on is distinguishable from one that passed (owner:unowned crit:low review:2027-01-09)
