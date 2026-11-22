---
slug: pose-validate-carries-unchanged-checks
status: draft
created_at: 2026-11-22
completed_at:
supersedes:
depends_on: pose-changed-scope-validation, pose-validation-evidence-material-binding
remediates:
priority: 1
components: pose-mcp
task_type: feature
surface: standard
delivers:
---

# Spec: pose-validate-carries-unchanged-checks

## 1. Intent

### Goal
Let `pose validate` refresh the canonical structured result by running only the checks a change can affect and carrying forward, unchanged and labelled, the results of the checks it cannot affect.

### Business value
In the Harne8 batch of 2026-11-22, every material change to one module required the whole 72-check matrix again. Each run took about 35 minutes with three jobs. Intermittent failures in unrelated suites made roughly one run in three useless. A hooks change confined to `scripts/` needed four full runs in one review round. The canonical record is all-or-nothing today. `--changed-from` already selects the affected modules, but it writes a partial record that cannot replace the canonical one in `results_path`, so closeout still needs the full run.

### Constraints
A carried result must say where it ran. It never upgrades its outcome, so a carried failure stays a failure. Any doubt about eligibility widens to a full run, and the record states why. The selection reuses the changed-scope rules and widens them only by explicit declaration. Results and indexes written by earlier engines keep their meaning.

### Non-goals
Caching individual command outputs, hashing arbitrary inputs of a check, carrying across a matrix or engine change, and changing how review bundles seal evidence beyond reading the per-check head.

## 2. Requirements

### Functional
- R1: `pose validate --carry --json-out <results_path>` shall read the canonical record at the configured `results_path` as its base. It shall be refused with exit 2 when no `results_path` is configured, when the output path is not that path, or when combined with `--module`, `--stack`, `--workspace`, `--changed-from` or `--changed-to`.
- R2: The base shall be used only when every condition below holds. Otherwise the run shall execute every check and record `carry.used: false` with the first failing condition as `carry.reason`.
  - Its schema, mode, `matrix_sha256` and `engine_version` equal the current run's.
  - Its `git_head` is an ancestor of, or equal to, `HEAD`.
  - It was not cancelled.
  - The working copy is clean in material paths.
- R3: With a usable base, the run shall select modules with the changed-scope rules over `base.git_head..HEAD`, excluding the `results_path` file itself. It shall execute every check of a selected module. Every check of an unselected module whose base result exists and was not skipped shall be carried. A module with any check lacking such a base result shall be selected with the reason `carry: no prior result for <check id>`.
- R4: A carried check shall keep the base result's outcome, exit code, duration, limit state, output and evidence class. It shall add `carried_from`, the head the check actually ran at, preserved through chained carries, and `carried_generated_at`. Counts shall add `carried`. Carried failures and errors shall count toward the run outcome exactly like executed ones.
- R5: The record shall add `engine_version` on every run. With `--carry` it shall also add a `carry` object: `used`, `reason`, `base_head`, `changed` (the number of files) and `selected` (module to reason).
- R6: A module declared `"selfContained": true` in `.pose/indexes/module-metadata.json` shall not be selected by a change outside every module. It is still selected by changes inside it, by its declared `dependsOn` and by policy. This shall hold for `--changed-from` and `--carry` alike, with the reason recorded for the modules it widens.
- R7: Reading the record into the delivery index shall use a check's `carried_from` and `carried_generated_at` as that result's head and time. Every consumer of a result's head therefore judges the commit a carried check really ran at.

### Non-functional
- R8: A run where every module is carried shall execute no check and finish in the time of the selection and the record write.

### Security
- R9: Revisions read from the base record shall pass the same confinement as `--changed-from` revisions before any Git call.

### Compatibility
- R10: A base record without `engine_version` shall never be used. Records without the new fields shall be read exactly as before.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/cli` (argument parsing, the run loop and record fields of `validate`, changed-scope selection), `pose-mcp/internal/pose` (reading the structured result into delivery results), the manual and the CLI help.

### Artifacts
- created: .pose/specs/2026-11-22-pose-validate-carries-unchanged-checks.md
- created: .pose/adr/2026-11-22-validate-carries-unchanged-checks.md
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/cli/validate_results.go
- created: pose-mcp/internal/cli/validate_carry.go
- created: pose-mcp/internal/cli/validate_carry_test.go
- modified: pose-mcp/internal/pose/delivery_validation_run.go
- created: pose-mcp/internal/pose/delivery_validation_run_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

### API/contract changes
Additive fields in the structured validation result: `engine_version`, `carry`, `counts.carried`, and per check `carried_from` and `carried_generated_at`. There is a new `validate --carry` flag and a new optional `selfContained` field in module metadata.

### Data/storage changes
None beyond the additive record fields.

### Technical risks
An instance that marks a module `selfContained` while its checks read root files would carry stale results. The declaration is explicit, documented and opt-in, and the default stays the full fallback.

## 4. Tasks

### Planning
- [ ] ADR for carrying results and the self-contained declaration.

### Implementation
- [ ] Record fields and base eligibility.
- [ ] Selection excluding the results file, the self-contained declaration and the no-prior-result widening.
- [ ] Carried entries in the run loop and counts.
- [ ] Per-check head in the delivery reading.
- [ ] Manual and help.

### Validation
- [ ] Unit and fixture tests for every requirement, each shown failing before its code.
- [ ] `pose validate --module pose-mcp` and `check --strict`.

## 5. Decisions

### Decision D1
A carried result is the base's result, not a new observation. It keeps the head where it ran, and consumers judge that head. The run's material binding is still computed at `HEAD`. This holds only because the selection guarantees that a carried module's inputs did not change between the two heads, under the changed-scope rules and the instance's declarations. See ADR `2026-11-22-validate-carries-unchanged-checks`.

### Decision D2
Eligibility compares the whole matrix digest and the engine version instead of per-check definitions. A matrix or engine change reruns everything. That is the existing rule for material bindings, and it keeps the comparison obviously sound.

### Decision D3
Out-of-module changes keep selecting every module unless a module declares itself self-contained. Instances opt in per module, so a repository that never declares anything gets today's behaviour.

## 6. Validation

### Strategy
Go tests in `internal/cli` drive real Git fixtures through `validate --carry`. They cover:
- the base conditions of R2, one fixture each;
- a change in one module that carries the other;
- chained carries;
- a carried failure;
- the self-contained declaration of R6, under both flags;
- the refusals of R1;
- the per-check head reading of R7.

Each test is run against the code before its requirement and must fail there.

### Deterministic checks

#### Test
- Command: `go test ./internal/cli/ ./internal/pose/` (in `pose-mcp`), then `pose validate --module pose-mcp`.

#### Lint
- Command: `go vet ./...` (in `pose-mcp`).

#### Typecheck
- Command: compiled by the tests.

#### Build
- Command: `go build ./...` (in `pose-mcp`).

#### Security / Contract
- Command: `pose check --strict`.

### Execution log

### Results summary

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

### Files and modules changed

### Validation executed

### Residual risks

### Follow-ups
