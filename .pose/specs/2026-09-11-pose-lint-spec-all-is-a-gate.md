---
slug: pose-lint-spec-all-is-a-gate
status: draft
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-one-follow-up-format
priority: 0
components: pose-mcp
task_type: refactor
delivers:
---

# Spec: `pose lint-spec --all` passes here, and stays passing

## 1. Intent

### Goal
`pose lint-spec --all` shall exit 0 on this repository, and something shall keep
it there.

### Business value
`pose lint-spec --all` exits 1 on this repository, on 5.0.1 and on main. Every
error comes from one spec, `pose-manual-and-cli-command-parity` (done
2026-08-21), which uses section names outside the template — "4. Artifacts",
"5. Verification Plan", "6. Delivery Evidence" — so the lint finds no Tasks, no
Validation and no requirement trace where it looks, and its surface target
`surface:cli-manual-parity` has no satisfied requirement carrying integration or
e2e evidence.

Nothing noticed, because nothing runs the command: CI does not, and `pose check
--strict` does not lint each spec's structure. So the one command that checks
every spec against the template cannot be used as a gate — the first time
anyone did, it failed for a reason unrelated to their change. That is how the
misplaced follow-up ownership of `pose-one-follow-up-format` went unseen too:
the lint that would have flagged it was never run over the repository.

### Constraints
- A done spec's recorded history is not rewritten: restructuring moves its
  content under the template's headings and adds what the template requires
  from evidence that already exists.

### Non-goals
- Changing the template or the lint's rules to accept this spec's headings.

---

## 2. Requirements

### Functional
- R1: `pose-manual-and-cli-command-parity` shall carry Tasks, Validation and a
  requirement trace under the template's headings, built from its own content
  and the commits that delivered it.
- R2: Its surface target shall trace to a satisfied requirement with the
  evidence class the surface requires, or the target shall be corrected.
- R3: `pose lint-spec --all` shall exit 0 on this repository.
- R4: A change that makes it fail again shall fail somewhere before merge
  (Decision 1).

### Non-functional
- The restructured spec keeps every statement it made.

---

## 3. Technical Plan

### Affected areas
- `.pose/specs/2026-08-21-pose-manual-and-cli-command-parity.md`
- the enforcement point chosen in Decision 1

### Artifacts
- created: .pose/specs/2026-09-11-pose-lint-spec-all-is-a-gate.md

### Technical risks
- The spec is `done`; if it carries an amendment log, restructuring must be
  acknowledged there or the amendment gate rejects it.

---

## 4. Tasks

### Implementation
- [ ] Increment 1: Restructure the spec from its own content and history (R1, R2)
- [ ] Increment 2: Enforce `lint-spec --all` where Decision 1 says (R3, R4)

### Validation
- [ ] Break a spec's structure on a branch and see the enforcement fail

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: where `lint-spec --all` is enforced.
- Options:
  - A. A step in this repository's CI. Protects this repository, changes
    nothing for instances, costs one workflow line.
  - B. `pose check --strict` lints every spec's structure. Protects every
    instance, but is a breaking change for any instance with a malformed done
    spec — which this repository shows is plausible — and belongs in a major
    release with an adoption note.
  - C. `pose doctor` reports specs that fail the lint, as a warning. Visible in
    every instance and never blocking, but a warning is what let this go unseen.
- Recommendation: A now, and B considered separately for a major release — it is
  the engine's contract, and deserves its own decision and ADR.
- Status: pending — the owner decides before Increment 2.

---

## 6. Validation

### Strategy
Run the lint over the whole repository before and after, and prove the chosen
enforcement fails on a deliberately broken spec.

### Deterministic checks

#### Lint
- Command: `pose lint-spec --all`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: local, engine at main
- Notes: draft. Measured: four errors, all in
  `pose-manual-and-cli-command-parity`; CI has no `lint-spec` step; `pose check
  --strict` does not lint spec structure.

### Results summary
- Successes: none yet.
- Failures: none.

### Requirement trace
- R1 [waived: draft, not implemented] <deferred from 5.0.2>
- R2 [waived: draft, not implemented] <deferred from 5.0.2>
- R3 [waived: draft, not implemented] <deferred from 5.0.2>
- R4 [waived: draft, not implemented] <pending Decision 1>

### Known gaps
- Everything; this spec records the defect, its cause and the decision it needs.

---

## 7. Final Report

### Summary
Deferred from 5.0.2 with the defect measured and its enforcement decision stated.

### Follow-ups
