---
slug: pose-lint-spec-all-is-a-gate
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-one-follow-up-format
priority: 0
components: pose-mcp
task_type: refactor
delivers: capability:lint-spec-all-gate
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
- created: .pose/changelogs/unreleased/pose-lint-spec-all-is-a-gate.md
- created: pose-mcp/internal/version/lint_spec_all_gate_test.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: .github/workflows/ci.yml
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-08-21-pose-manual-and-cli-command-parity.md
- modified: .pose/specs/2026-09-20-check-builds-the-delivery-graph-once.md

- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_instance_config_test.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/install_locale_identity_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/managed_docs_test.go
- created: pose-mcp/internal/cli/mcp_sigterm_test.go
- modified: pose-mcp/internal/cli/release_compatibility_test.go
- modified: pose-mcp/internal/cli/self_update_release_test.go
- modified: pose-mcp/internal/cli/stack_seed.go
- created: pose-mcp/internal/cli/update_reports_delivered_state_test.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_root_and_nodemodules_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/pose/discovery.go

Backfilled on 2026-09-29: `20c6565` (Before 5.0.2: honest update failures, SIGTERM on stdio, and the overdue backlog (#94)) carries this spec's trailer, because it created this spec as a draft, and also changed the 18 paths declared last. They are claimed so the Git change set reconciles; they record provenance, not this spec's design.

### Delivery targets
- capability:lint-spec-all-gate module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

The CLI package changes only by a test, so the target is a capability; the
gate itself is a CI step.

### Technical risks
- The spec is `done`; if it carries an amendment log, restructuring must be
  acknowledged there or the amendment gate rejects it.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Restructure the spec from its own content and history (R1, R2)
- [x] Increment 2: Enforce `lint-spec --all` where Decision 1 says (R3, R4)

### Validation
- [x] Break a spec's structure and see the enforcement fail
- [ ] Run the checks, obtain review and close

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
- Status: decided 2026-09-29 by the owner: option A. B stays a separate
  decision for a major release.

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

- Date: 2026-09-29
- Environment: local, engine at main
- Notes: before this work, 2 specs failed: `pose-manual-and-cli-command-parity`
  (Tasks and Validation missing, trace outside Validation, surface without
  integration evidence) and `check-builds-the-delivery-graph-once` (`[covered]`
  without a slug). A closeout batch the same day added 38 more, all surface
  targets without a traced integration requirement, and no gate reported it;
  they were corrected before this spec's increments. Restructuring the parity
  spec found that its trace cited `TestReleaseInputs`, which never existed, and
  mixed `check:unit` with `evidence:integration`; R5 now cites a regression test
  that fails with the pre-`bd0636f` lookup, and the target was corrected to a
  capability. After both increments `pose lint-spec --all` exits 0. With the
  `[covered]` slug removed again it exits 1, which is what the new CI step runs;
  the workflow test fails when the step is absent. Both edited done specs keep
  their approved reviews (retained completed scope).

### Results summary
- Successes: R1 to R4.
- Failures: none.

### Requirement trace
- R1 [satisfied] capability:lint-spec-all-gate evidence:integration check:lint-spec-all-gate-contract test:TestReleasePrepareFindsADatePrefixedSpec — the parity spec carries Tasks, Validation and a trace under the template headings, each line naming evidence that exists
- R2 [satisfied] capability:lint-spec-all-gate evidence:manual — the surface target was corrected to capability:cli-manual-parity, since its evidence is unit-level file parity
- R3 [satisfied] capability:lint-spec-all-gate evidence:manual — `pose lint-spec --all` exits 0 on 2026-09-29, with 0 failed specs
- R4 [satisfied] capability:lint-spec-all-gate evidence:integration check:lint-spec-all-gate-contract test:TestCIGovernanceJobRunsLintSpecAll — CI's governance job runs the lint, and the test fails if the step is removed

### Known gaps
- Instances are not protected: option B (lint in `check --strict`) needs a major
  release and an ADR.

---

## 7. Final Report

### Summary
`pose lint-spec --all` exits 0 on this repository and runs in CI, so a spec
that breaks the template fails before merge.

### Follow-ups
- [open] Decide in the next major, with an ADR, whether `pose check --strict` lints every spec against the template for all instances (Decision 1, option B) (owner:@pose-maintainers crit:medium review:2026-12-29)
