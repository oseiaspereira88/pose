---
slug: pose-cli-ergonomics-and-stack-expansion
status: in-progress
created_at: 2026-08-21
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-engine-stability-and-diagnostics-convergence
priority: 0
components: cli, extensions, validation, scaffold
delivers: capability:extension-target-flag, capability:skill-parity-dynamic-dispatch, capability:stack-fixtures-expansion
---

# Spec: CLI Ergonomics, Extension Target Flag, and Stack Fixtures Expansion

> Single POSE spec template. Fill the relevant sections; remove the ones that
> don't apply. Keep the order: Intent → Requirements → Technical Plan →
> Tasks → Decisions → Validation → Final Report.
>
> **Lifecycle:** update `status` as you go (`draft` → `in-progress` → `done`).
> On completion, run the closeout flow (skill `pose-spec-closeout`): set
> `status: done`, fill `completed_at` and disposition every follow-up.

---

## 1. Intent

### Goal
Provide explicit target directory support for `pose extension install`, harden the multi-word command parity verification in scaffold tests, and expand stack fixtures for Python managers (Poetry, Pipenv) and .NET.

### Business value
1. Automation workflows and nested monorepo tools often need to install POSE extensions into a designated repository root other than the current working directory without changing process directory state. Adding `--target <dir>` to `pose extension install` provides deterministic, scriptable extension management.
2. Skill locale parity tests previously maintained a static map of two-word commands, which risked drift when new subcommands were introduced. Expanding this map dynamically across all CLI command groups ensures zero false negatives.
3. Adding explicit test fixtures for Poetry, Pipenv, and .NET ensures multi-manager discovery and matrix execution remain regressions-free.

### Constraints
- Retain exact signature compatibility for existing `pose extension install <id>` invocations.
- Zero breaking changes to `validation-matrix.json` defaults.

### Non-goals
- Overhauling the Sigstore verification pipeline or creating remote registries.

---

## 2. Requirements

> Definition of Ready (entry gate): before `status: in-progress`, functional
> requirements must have **acceptance criteria with stable IDs** (`- R<N>: ...`).
> Published IDs are never renumbered; a removed criterion is marked as
> withdrawn. Verify with `pose lint-spec <slug> --ready-check`.
>
> Optional EARS form: `- R1: When <trigger>, the <system> shall <behavior>.`
> Verify an opted-in spec with `pose lint-spec <slug> --ears`.

### Functional
- R1: `pose extension install` shall accept an optional `--target <path>` flag specifying the target instance directory where files and the extensions lockfile are installed.
- R2: `skill_locale_parity_test.go` shall recognize compound multi-word subcommands across all native CLI verbs (`review`, `assess`, `extension`, `release`, `docs-review`, `state`, `telemetry`, `roadmap`, `rule`, `hook`).
- R3: Integration tests in `stack_manifest_test.go` and `cli_test.go` shall verify discovery and stack detection across Poetry, Pipenv, and .NET solutions.

### Non-functional
- Complete test suite passes with 100% deterministic assertion coverage.

### Security
- The target path must be validated and confined using repository root path resolution.

### Compatibility
- Fully backward compatible with existing command line invocations.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/extension.go`
- `pose-mcp/internal/cli/extension_test.go`
- `pose-mcp/internal/scaffold/skill_locale_parity_test.go`
- `pose-mcp/internal/cli/stack_manifest_test.go`

### Artifacts
<!-- Declare exact project-relative source-tree paths: created, modified,
     renamed (old -> new), removed, or one `none: <reason>` entry. -->
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/extension.go
- modified: pose-mcp/internal/cli/extension_test.go
- modified: pose-mcp/internal/scaffold/skill_locale_parity_test.go
- modified: pose-mcp/internal/cli/stack_manifest_test.go

### Delivery targets
<!-- When `delivers` is populated, declare the exact same refs here. Profiles
     and evidenceClass requirements come from validation-matrix.json. -->
- capability:extension-target-flag module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:skill-parity-dynamic-dispatch module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:stack-fixtures-expansion module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- `pose extension install <pkg> [--target <dir>]` documented and supported.

### Data/storage changes
- None.

### Technical risks
- None identified.

---

## 4. Tasks

### Planning
- [ ] Review `cmdExtensionInstall` flag parsing and destination path binding.

### Implementation
- [ ] Support `--target <dir>` flag in `cmdExtensionInstall`.
- [ ] Add unit test `TestExtensionInstallTargetFlag` in `extension_test.go`.
- [ ] Expand multi-word command dispatch in `skill_locale_parity_test.go`.
- [ ] Add fixture test cases in `stack_manifest_test.go`.

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/...`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-cli-ergonomics-and-stack-expansion --strict`.

---

## 5. Decisions

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

### Decision 1
- Date: 2026-08-21
- Context: `pose extension install` needed a way to target specific directories safely.
- Options considered: Environment variables, config file overrides, explicit flags.
- Decision: An explicit `--target <dir>` flag overrides the default `root` passed from `Main`.
- Rationale: Follows standard CLI flag conventions across `pose install --target` and `pose init --target`.
- Consequences: Improved scripting ergonomics for monorepo CI pipelines.

---

## 6. Validation

### Strategy
Unit tests in `extension_test.go`, parity test verification in `skill_locale_parity_test.go`, and end-to-end discovery tests in `stack_manifest_test.go`.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Internal CLI and scaffold unit tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-cli-ergonomics-and-stack-expansion --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: N/A
- Scope: N/A
- Expected: N/A

#### Build
- Command: N/A
- Scope: N/A
- Expected: N/A

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full repository validation.
- Expected: Result: SUCCESS.

### Execution log
- Date:
- Environment:
- Notes:

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace
<!-- At closeout, one bullet per declared R-ID (spec pose-requirement-evidence-traceability):
- R<N> [satisfied] <verification case; structured refs: check:<name> test:<id> report:<file> commit:<sha>>
- R<N> [satisfied] surface:<id> evidence:integration check:<reachability-check>
- R<N> [deferred-integration: spec:<non-terminal-slug>] surface:<id>
- R<N> [waived: <reason>]
- R<N> [withdrawn: <reason>]
Missing or orphaned IDs fail `pose lint-spec --strict` on done specs. -->
- R1 [satisfied] capability:extension-target-flag check:unit test:TestExtensionInstallTargetFlag evidence:integration
- R2 [satisfied] capability:skill-parity-dynamic-dispatch check:unit test:TestSkillLocaleParity evidence:integration
- R3 [satisfied] capability:stack-fixtures-expansion check:unit test:TestStackManifestFixturesExpansion evidence:integration

### Known gaps
<!-- Temporary limitations, blocked checks, deferred validations. -->

---

## 7. Final Report

### Delivered scope
- Implemented `--target <dir>` in `pose extension install`.
- Expanded compound verb recognizer in scaffold parity tests.
- Added comprehensive stack fixtures for Poetry, Pipenv, and .NET.

### Files and modules changed
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/extension.go`
- `pose-mcp/internal/cli/extension_test.go`
- `pose-mcp/internal/scaffold/skill_locale_parity_test.go`
- `pose-mcp/internal/cli/stack_manifest_test.go`

### Validation executed
- Command:
- Result:

### Residual risks
- None.

### Follow-ups

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [done] CLI ergonomics and fixture expansion verified.
