---
slug: pose-review-plan-producible-evidence-classes
status: in-progress
created_at: 2026-09-07
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
delivers:
---

# Spec: Review plans may only demand producible evidence

## 1. Intent

### Goal
Stop the review planner from requiring evidence classes that no registered
check is permitted to emit, so that a manual attestation can be recorded
truthfully instead of `pose review auto-attest` being the only path that
completes.

### Business value
`buildReviewTools` synthesised a `validate` tool per component with a literal
evidence class:

```go
add("validate", "required", component.Path, []string{"validation"}, nil)
```

`ValidEvidenceClasses` does not contain `validation`, and `pose validate`
rejects any check declaring it — so no evidence of that class can exist in any
project. `cmdReviewAttest` enforces the class on the disposition, and refuses:

```
pose review attest: review tool validate (component site) requires evidence
class validation
```

There is no truthful value to supply. The ref is not cross-checked against the
bundle's evidence either, so the two ways through were to assert a class the
check does not have, or to run `pose review auto-attest`, which fills the gap
with `<class>:auto-attest`.

That is the sharp end of it: for any scope with mapped components, the only
mechanism that completed a review was the fabricating one. A gate whose sole
viable path invents its own evidence is worse than a gate that fails, because
it reports green while proving nothing — and the fabricated attestations it
produces are indistinguishable, in the record, from reviewed ones.

The literal also defeated the workaround a project could otherwise use. A
project that owns its review profiles and reconciles the classes there still
received component-scoped tools carrying `validation`, because the class never
came from the profile.

### Constraints
- Do not widen `ValidEvidenceClasses`. Accepting `validation` would admit a
  class that still has no producer, which is how the divergence began.
- A dropped class must be visible. Silently narrowing a gate is the same
  failure in the other direction.

### Non-goals
- Reconciling the evidence classes the shipped profiles declare on their
  *criteria*. Those already degrade to a real evidence ref in a manual
  attestation and do not block it.

---

## 2. Requirements

### Functional
- R1: A synthesised review tool shall take its evidence classes from the
  profile that governs it, never from a literal in the planner.
- R2: A review plan shall not carry an evidence class outside
  `ValidEvidenceClasses` on any tool.
- R3: A class dropped for being unproducible shall surface as a plan warning
  naming the tool and the class.

### Non-functional
- The existing suite passes unchanged in behaviour.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_plan.go` — `buildReviewTools`

### Artifacts
- created: .pose/specs/2026-09-07-pose-review-plan-producible-evidence-classes.md
- renamed: .pose/changelogs/unreleased/pose-review-plan-producible-evidence-classes.md -> .pose/changelogs/v1.7.12/pose-review-plan-producible-evidence-classes.md
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_plan_test.go

### Technical risks
- Dropping a class relaxes a tool that previously could not be satisfied at
  all, so nothing that used to pass starts failing. The reverse — a project
  that leaned on the impossible constraint to keep a tool unsatisfiable — is not
  a use case worth preserving.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Take the synthesised validate tool's classes from the profile (R1)
- [x] Increment 2: Filter every tool's classes to what a check may emit (R2)
- [x] Increment 3: Warn on each dropped class (R3)

### Validation
- [x] Suite verde; regressão vermelha sem o fix

---

## 5. Decisions

### Decisão 1
- Date: 2026-09-07
- Context: where the class constraint on a synthesised tool should come from.
- Options considered: (a) derive it from the checks the module actually
  registers; (b) take it from the governing profile; (c) drop the constraint.
- Decision: (b), with the filter from R2 on top.
- Rationale: (a) is the most faithful but needs the validation matrix threaded
  into a planner that today has no Store and is unit-tested without one — a
  larger change than the defect warrants. (c) alone would let a profile's
  legitimate constraint be lost. (b) keeps the profile authoritative, which is
  where a project can already reconcile, and the filter keeps an unreconciled
  profile from reintroducing an impossible demand.
- Consequences: a shipped profile that still names `validation` no longer makes
  attestation impossible; the class is dropped and the drop is reported.

---

## 6. Validation

### Strategy
Assert the invariant directly — no tool in a planned review may name a class
outside `ValidEvidenceClasses` — and prove the assertion fails without the fix.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Regression
- Command: `go test ./internal/pose/ -run TestReviewPlanToolsOnlyDemandProducibleEvidenceClasses`
  with the literal restored and the filter disabled
- Expected: fails naming `validate (component "api")` and `validation`

### Execution log
- Date: 2026-09-07
- Environment: local, Go 1.26
- Notes: all eight packages pass. Reintroducing the literal reproduces the
  defect verbatim: `tool validate (component "api") demands evidence class
  "validation", which no check may emit`. The regression also asserts the
  fixture actually produces a component-scoped validate tool, so the test
  cannot pass by observing nothing.
  `TestReviewCheckRequiresCurrentEffectivePlanDigestAndCoverage` needed its
  fixture decoupled: it hardcoded `evidence:validation:review-plan` and built
  the string from the first evidence class, so it was asserting through the very
  literal being removed. It now captures the exact line it wrote for
  `validate (component api)` — other tools render an identical suffix, so the
  whole line is the only unambiguous handle.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <review_plan.go — profileEvidence("validate") replaces the literal at both synthesis sites>
- R2 [satisfied] <review_plan.go — producible() filters in add(), so every tool is covered, not just validate; asserted by TestReviewPlanToolsOnlyDemandProducibleEvidenceClasses>
- R3 [satisfied] <a dropped class appends "review tool X drops evidence class Y: no registered check may emit it" to plan warnings, threaded through buildReviewTools into plan.Warnings>

### Known gaps
- The criteria side still names classes with no producer (`test`,
  `requirement-trace`). It does not block a manual attestation — that path falls
  back to a real evidence ref — but it does drive `auto-attest` output.

---

## 7. Final Report

### Follow-ups

- [open] Apply the same producible filter to criterion evidence classes so auto-attest stops inventing refs — owner:unowned crit:medium review:2026-12-07
- [open] Derive tool evidence classes from the module's registered checks once the planner can reach the matrix — owner:unowned crit:low review:2026-12-07
