---
slug: pose-cli-universal-help-and-subcommand-introspection
status: done
created_at: 2026-08-21
completed_at: 2026-08-21
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-contributor-mode-protocol
priority: 0
components: cli, docs
delivers: capability:cli-universal-help, capability:subcommand-help-introspection, capability:bilingual-cli-help
---

# Spec: CLI Universal Help and Subcommand Introspection

## 1. Intent

### Goal
Provide universal, first-class `-h` and `--help` flag support across all POSE CLI commands and subcommands, hierarchical `pose help <command> [subcommand]` introspection, and rich bilingual documentation (synopsis, flags, examples) with standard POSIX exit code `0`.

### Business value
1. **Agent Self-Sufficiency**: AI agents frequently probe command capabilities using `pose <cmd> -h` or `pose <cmd> <subcmd> --help`. When commands fail with unhandled flag errors (exit code 2), agents fail or search external files. Providing deterministic, structured CLI help keeps agents autonomous and accurate.
2. **Developer Ergonomics**: Human developers navigating the terminal expect standard `-h`/`--help` flags on every sub-action without having to consult external markdown files for routine flag lookups.
3. **Consistency & Standards**: Conforms to standard CLI ergonomics across all 50+ commands, gates, workflows, and extension mechanisms.

### Constraints
- Every command and subcommand must return exit code `0` on `-h` and `--help`.
- 100% linguistic parity between English (`en`) and Portuguese (`pt-BR`).
- Zero regression on existing command argument parsing.

### Non-goals
- Introducing external CLI framework dependencies; implementation remains in native Go stdlib.

---

## 2. Requirements

### Functional
- R1: Implement global `-h` and `--help` flag interception across all CLI commands and subcommands, ensuring exit code `0` and structured help output.
- R2: Support hierarchical help queries via `pose help <command> [subcommand]` with identical output to `pose <command> [subcommand] --help`.
- R3: Author comprehensive help definitions for all registered commands and subcommands (synopsis, descriptions, flags, examples).
- R4: Provide 100% bilingual parity for all help entries based on the active CLI locale (`POSE_LOCALE` / `LANG`).
- R5: Deliver automated test suite in `help_test.go` verifying `-h`, `--help`, and `pose help <cmd>` across all commands.

### Non-functional
- Complete deterministic verification via `go test ./...` and `pose validate --strict`.

### Security
- No sensitive data or environment secrets exposed in help texts.

### Compatibility
- Fully backward compatible with all existing commands and automation scripts.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/help.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `pose-mcp/internal/cli/help_test.go`
- `pose-mcp/internal/cli/cli.go`

### Artifacts
- created: pose-mcp/internal/cli/help.go
- created: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/cli/help_test.go
- modified: pose-mcp/internal/cli/cli.go

### Delivery targets
- capability:cli-universal-help module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:subcommand-help-introspection module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:bilingual-cli-help module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- All commands and subcommands accept `-h` and `--help` returning exit code `0`.
- `pose help <cmd> [subcmd]` outputs command-specific help.

### Data/storage changes
- None.

### Technical risks
- None.

---

## 4. Tasks

### Planning
- [ ] Catalog all CLI commands and subcommands with their flags and descriptions.

### Implementation
- [ ] Author `pose-mcp/internal/cli/help_catalog.go` containing structured bilingual metadata for all commands and subcommands.
- [ ] Author `pose-mcp/internal/cli/help.go` with `dispatchHelp`, `hasHelpFlag`, and formatting utilities.
- [ ] Wire help interception in `pose-mcp/internal/cli/cli.go`.
- [ ] Author exhaustive test suite in `pose-mcp/internal/cli/help_test.go`.

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/cli -run Help`.
- [ ] Run `go test -v ./pose-mcp/internal/...`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-cli-universal-help-and-subcommand-introspection --strict`.

---

## 5. Decisions

### Decision 1: Pre-Execution Help Interception
- Date: 2026-08-21
- Context: Individual commands handle flag parsing differently (some use standard `flag` package, some manual slices).
- Options considered: Middleware vs per-command logic.
- Decision: Intercept `-h` and `--help` at the top-level command router (`cli.go`) and within subcommand routers before executing business logic.
- Rationale: Guarantees universal coverage without modifying dozens of individual command parsing loops and prevents side effects during help inspection.
- Consequences: Cleaner command definitions but requires a centralized routing registry.

---

## 6. Validation

### Strategy
Unit tests across all commands testing `-h`, `--help`, and `pose help <cmd> [subcmd]`.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/cli -run Help`
- Scope: Universal help tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-cli-universal-help-and-subcommand-introspection --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Project type check.
- Expected: Pass.

#### Build
- Command: `go build ./pose-mcp/...`
- Scope: Project build.
- Expected: Success.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
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
- R1 [satisfied] capability:cli-universal-help check:unit test:TestUniversalHelpFlagsAllCommands evidence:integration
- R2 [satisfied] capability:subcommand-help-introspection check:unit test:TestHierarchicalHelpCommand evidence:integration
- R3 [satisfied] capability:cli-universal-help check:unit test:TestHelpCatalogCompleteness evidence:integration
- R4 [satisfied] capability:bilingual-cli-help check:unit test:TestBilingualHelpParity evidence:integration
- R5 [satisfied] capability:cli-universal-help check:unit test:TestUniversalHelpFlagsAllCommands evidence:integration

### Known gaps
- None.

---

## 7. Final Report

### Delivered scope
- Implemented universal help dispatcher and command catalog in `help.go` and `help_catalog.go`.
- Supported `-h` and `--help` on all 50+ commands and subcommands with exit code 0.
- Implemented hierarchical `pose help <cmd> [subcmd]` dispatch.
- Added comprehensive bilingual tests in `help_test.go`.

### Files and modules changed
- `pose-mcp/internal/cli/help.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `pose-mcp/internal/cli/help_test.go`
- `pose-mcp/internal/cli/cli.go`

### Validation executed
- Command: `go test -v ./pose-mcp/internal/cli -run Help`
- Result: Pass

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

- [done] Universal CLI help implemented and tested across all commands and subcommands.
