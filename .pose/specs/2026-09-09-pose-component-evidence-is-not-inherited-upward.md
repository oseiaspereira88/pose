---
slug: pose-component-evidence-is-not-inherited-upward
status: in-progress
created_at: 2026-09-09
completed_at:
supersedes:
depends_on: pose-evidence-scoped-to-component
priority: 0
components: pose-mcp
delivers:
---

# Spec: Component evidence answers downward, not upward

## 1. Intent

### Goal
Stop a result from a directory inside a component from satisfying a criterion or
delivery target about the whole component.

### Business value
`pose-evidence-scoped-to-component` refused evidence from a sibling: real, of the
demanded class, and silent about the thing the criterion was about. It matched by
`moduleMatchesTarget`, which accepted a path prefix in **either** direction — so
the sibling case closed while the containment case stayed open in both readings.

One of the two is sound. A result registered for a module answers for a target
inside it, because a module-wide run — `go test ./...`, `npm test` — exercises
its subtree, and running checks once at the module root is how nearly every
project is laid out.

The reverse does not follow. A result from `site/api` covers one directory of
`site` and was accepted as covering all of it. A component could be gated on
evidence that never touched most of it, which is the same defect the sibling
refusal exists to prevent, one relation over.

### Constraints
- The repository root must keep answering for everything and being answered by
  anything. A single-module project is its own component.
- The downward direction must keep working, or the ordinary layout becomes
  unsatisfiable.

### Non-goals
- Requiring an exact module match. POSE would then be inferring a stricter claim
  than the result makes in the downward direction too, and the layout where a
  module runs its checks once would stop closing.
- Making a check declare the subtree it actually walked. That is the durable
  answer to both directions and a much larger change.

---

## 2. Requirements

### Functional
- R1: A result registered for a module shall answer for a target or criterion
  inside that module.
- R2: A result registered for a directory inside a component shall not answer
  for the component.
- R3: Siblings shall continue not to answer for each other, and a name that
  merely shares a textual prefix shall not count as inside.
- R4: The repository root shall keep answering for everything, and a target
  declared at the root shall keep being answered by anything.

### Non-functional
- The rule is exercised where it is consumed, not only in isolation.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/delivery_surface.go` — `moduleMatchesTarget`, the one
  place all six call sites share

### Artifacts
- created: .pose/specs/2026-09-09-pose-component-evidence-is-not-inherited-upward.md
- renamed: .pose/changelogs/unreleased/pose-component-evidence-is-not-inherited-upward.md -> .pose/changelogs/v4.0.0/pose-component-evidence-is-not-inherited-upward.md
- created: pose-mcp/internal/pose/module_scope_direction_test.go
- modified: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: .pose/specs/2026-09-09-pose-evidence-scoped-to-component.md

### Technical risks
- A component whose only evidence sits in a directory inside it now has none, so
  such a scope stops closing until a check is registered for the component or the
  target is declared where the evidence is. That is the intended refusal, and it
  is a new one.
- Nothing in this repository exercises either prefix direction, so the change
  cannot be validated against its own data. Fixtures carry it, and the
  `surface-check` edge count is the control that nothing here moved.

---

## 4. Tasks

### Implementation
- [x] Increment 1: `moduleMatchesTarget` answers downward only (R1, R2, R3, R4)

### Validation
- [x] The rule shown failing when the upward direction is restored

---

## 5. Decisions

### Decision 1
- Date: 2026-09-09
- Context: which of the two prefix directions is unsound. The proposal put to the
  maintainer named the wrong expression while describing the right case, and the
  described case turned out to be the one worth keeping.
- Decision: keep downward (a containing module answers), refuse upward (a
  contained directory does not). Recorded as an amendment to the sealed review
  bundles ADR.
- Rationale: the downward direction is a claim the result actually supports; the
  upward one is partial coverage read as complete. Measured before deciding:
  removing either direction leaves this repository's `surface-check` at the same
  524 `validated-by` edges, because its targets carry module `.` or `pose-mcp`
  and its results carry `pose-mcp` — so neither prefix fires here and the choice
  had to be made on the semantics, not on what happened to break.

---

## 6. Validation

### Strategy
Assert both directions directly and through `BuildDeliverySurface`, and require
the assertions to fail when the upward direction is put back.

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
- Notes: restoring `|| strings.HasPrefix(module, target+"/")` fails three of the
  new assertions — the two table rows for upward containment and the
  delivery-surface fixture. `surface-check` reports 524 `validated-by` edges and
  358 findings before and after, which is the control that this repository's own
  graph is unchanged.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestModuleMatchesTargetAnswersDownwardOnly covers one and two levels down; TestATargetInsideAModuleTakesTheModulesRun asserts it through BuildDeliverySurface>
- R2 [satisfied] <the same table refuses `site/api` for `site` and `site/api/handlers` for `site`; TestDeliveryTargetIsNotSatisfiedByEvidenceFromOneDirectoryInsideIt asserts it through BuildDeliverySurface, reading the `validated-by` edge rather than the absence of a finding>
- R3 [satisfied] <the table refuses `site/api` for `site/web`, and `site` for `sitemap/api` — a shared textual prefix is not containment>
- R4 [satisfied] <the table keeps `.`, `""` and `root` answering in both positions>

### Known gaps
- The rule is still inferred from paths. A check that walks only part of its
  module is accepted as answering for all of it, in the downward direction, and
  nothing declares what a check actually walked.

---

## 7. Final Report

### Summary
Evidence answers for the component that contains it, not for one that contains
the evidence.

### Follow-ups

- [open] Have a check declare the subtree it actually walked, so containment is read from the run rather than inferred from the module path — it is the durable answer to both directions, and the downward one is still an inference — owner:unowned crit:medium review:2027-01-09
