---
slug: pose-spec-format-migration-command
status: in-progress
created_at: 2026-08-21
completed_at:
supersedes:
depends_on: pose-specs-ergonomics-and-discovery
priority: 0
components: cli, docs
delivers: capability:spec-format-migration, capability:companion-artifact-preservation, capability:spec-format-help
---

# Spec: Spec Format Migration CLI Command Suite

## 1. Intent

### Goal
Provide a native `pose spec-format migrate <slug>|--all [--format folder|flat] [--dry-run]` CLI command to migrate legacy specification structures into date-prefixed chronological layouts (`YYYY-MM-DD-<slug>/spec.md` or `YYYY-MM-DD-<slug>.md`), enforcing mandatory directory envelope preservation whenever companion artifacts (e.g. `amendments.jsonl`, split section files) are present.

### Business value
1. **Effortless Repository Modernization**: Enables teams to transition existing repositories with dozens of legacy specs into clean chronological file trees with a single command (`pose spec-format migrate --all`).
2. **Safety & Data Preservation**: Strictly guarantees that specs with amendment histories (`amendments.jsonl`) are never flattened or stripped of their audit trails.
3. **Idempotence & Predictability**: `--dry-run` previews every file move, and already-migrated specs are detected and skipped without errors.

### Constraints
- Never flatten a spec that contains companion artifacts like `amendments.jsonl`.
- Frontmatter `slug` must remain identical and canonical.
- Full bilingual parity for CLI messages and universal `-h` / `--help` support.

### Non-goals
- Mutating markdown body text or altering requirement IDs.

---

## 2. Requirements

### Functional
- R1: Implement `pose spec-format migrate <slug>|--all [--format folder|flat] [--dry-run]` in `pose-mcp/internal/cli/spec_format.go`.
- R2: Derive the date prefix accurately from frontmatter `created_at` (falling back to `completed_at` or current date).
- R3: Enforce that specs with `amendments.jsonl` or split sections are always migrated into date-prefixed folder envelopes (`YYYY-MM-DD-<slug>/`), preserving all companion files.
- R4: Support `--dry-run` flag to display proposed migration plans without modifying files on disk.
- R5: Integrate structured bilingual help in `help_catalog.go` and ensure `-h` / `--help` return exit code `0`.
- R6: Provide exhaustive automated test coverage in `spec_format_test.go` verifying single migration, batch migration, amendment preservation, and dry-run mode.

### Non-functional
- Complete deterministic verification via `go test ./...` and `pose validate --strict`.

### Security
- Ensure all target paths remain confined to `.pose/specs/` without path traversal vulnerabilities.

### Compatibility
- 100% compatible with hybrid resolution in engine (`pose.Store.GetSpec`).

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/spec_format.go`
- `pose-mcp/internal/cli/spec_format_test.go`
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `docs-site/docs/cli.md`

### Artifacts
- created: pose-mcp/internal/cli/spec_format.go
- created: pose-mcp/internal/cli/spec_format_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md

### Delivery targets
- capability:spec-format-migration module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:companion-artifact-preservation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:spec-format-help module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- New CLI command: `pose spec-format <migrate|status> [<slug>|--all] [--format folder|flat] [--dry-run] [--json]`.

### Data/storage changes
- Renames legacy spec files/folders to date-prefixed naming conventions.

### Technical risks
- None.

---

## 4. Tasks

### Planning
- [ ] Align migration options and companion artifact invariants.

### Implementation
- [ ] Implement `cmdSpecFormat` in `pose-mcp/internal/cli/spec_format.go`.
- [ ] Wire command routing in `pose-mcp/internal/cli/cli.go`.
- [ ] Add structured bilingual help in `pose-mcp/internal/cli/help_catalog.go`.
- [ ] Author comprehensive unit tests in `pose-mcp/internal/cli/spec_format_test.go`.
- [ ] Update documentation manuals (`POSE.md`, `docs-site/docs/cli.md`).

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/cli -run SpecFormat`.
- [ ] Run `go test ./pose-mcp/...`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-spec-format-migration-command --strict`.

---

## 5. Decisions

### Decision 1: Mandatory Folder Envelope for Companion Artifacts
- **Date**: 2026-08-21
- **Context**: Users can request `--format flat` for monolithic specs, but some specs contain amendment event logs (`amendments.jsonl`) or split section files.
- **Decision**: If any companion files exist within the spec directory, ignore `--format flat` and force migration to a date-prefixed folder envelope (`YYYY-MM-DD-<slug>/`).
- **Rationale**: Prevents data loss and preserves the append-only audit trail required by POSE governance.

---

## 6. Validation

### Strategy
Unit tests in `cli` package testing single spec migration, `--all` batch migration, companion amendment preservation, dry-run safety, and error handling.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/cli -run SpecFormat`
- Scope: Spec format migration tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-spec-format-migration-command --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: `go test ./pose-mcp/...`
- Scope: Typecheck.
- Expected: Pass.

#### Build
- Command: `go build ./pose-mcp/...`
- Scope: Build.
- Expected: Success.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Requirement trace
- R1 [satisfied] capability:spec-format-migration check:unit test:TestSpecFormatMigrate_SingleAndAll evidence:integration
- R2 [satisfied] capability:spec-format-migration check:unit test:TestSpecFormatMigrate_DateDerivation evidence:integration
- R3 [satisfied] capability:companion-artifact-preservation check:unit test:TestSpecFormatMigrate_AmendmentsPreserved evidence:integration
- R4 [satisfied] capability:spec-format-migration check:unit test:TestSpecFormatMigrate_DryRun evidence:integration
- R5 [satisfied] capability:spec-format-help check:unit test:TestSpecFormat_Help evidence:integration
- R6 [satisfied] capability:spec-format-migration check:unit test:TestSpecFormatMigrate_SingleAndAll evidence:integration

### Known gaps
- None.

---

## 7. Final Report

### Delivered scope
- Implemented `pose spec-format migrate` and `pose spec-format status` CLI commands with `--all`, `--dry-run`, `--format folder|flat`, and `--json`.
- Implemented automatic date derivation from frontmatter.
- Enforced companion artifact directory preservation for specs with amendments or split sections.
- Added structured bilingual help and comprehensive automated tests.

### Files and modules changed
- `pose-mcp/internal/cli/spec_format.go`
- `pose-mcp/internal/cli/spec_format_test.go`
- `pose-mcp/internal/cli/cli.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `docs-site/docs/cli.md`

### Validation executed
- Command: `go test -v ./pose-mcp/internal/cli -run SpecFormat`
- Result: Pass

### Residual risks
- None.

### Follow-ups
- [done] Spec format migration CLI command suite implemented and verified.
