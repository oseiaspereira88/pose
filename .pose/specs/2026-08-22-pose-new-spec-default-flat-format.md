---
slug: pose-new-spec-default-flat-format
status: done
created_at: 2026-08-22
supersedes:
depends_on: pose-release-closeout-skill-policy-link
priority: 25
components: cli, scaffold, help
delivers: surface:new-spec-flat-default
completed_at: 2026-08-22
---

# Spec: Modern Dated Flat Layout as Default for new-spec

## 1. Intent

### Goal
Fix issue #33 so `pose new-spec <slug>` creates the canonical modern dated flat file `.pose/specs/YYYY-MM-DD-<slug>.md` by default, ensuring immediate format conformance (`pose spec-format status` reporting `conforming: true`) without requiring post-scaffold migration.

### Business value
Aligns runtime scaffolding with the modern POSE specification format standard and documentation, removing friction and confusion when creating new specs.

### Constraints
- Zero breaking changes: `--folder` / `--dated` creates date-prefixed folders for specs with amends; `--legacy` creates legacy non-dated folders.
- Complete parity across help text and localization.

### Non-goals
- Removing support for dated folders with amends.

---

## 2. Requirements

### Functional
- R1: `cmdNewSpec` shall create `.pose/specs/YYYY-MM-DD-<slug>.md` by default when no folder/legacy flag is provided.
- R2: `cmdNewSpec` shall support `--folder` (or `--dated`, `--dir`) to create `.pose/specs/YYYY-MM-DD-<slug>/spec.md` for specs that will contain amendments.
- R3: `cmdNewSpec` shall support `--legacy` to create `.pose/specs/<slug>/spec.md`.
- R4: `pose spec-format status --json` shall report `conforming: true` immediately after scaffolding a new spec.
- R5: CLI help catalog for `new-spec` shall document the modern dated default format and flags in English and Portuguese.

### Non-functional
- Complete test coverage in `internal/cli`.
- Clean validation across all modules.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/scaffold.go`
- `pose-mcp/internal/cli/help_catalog.go`
- `pose-mcp/internal/cli/specs_cmd_test.go`
- `pose-mcp/internal/cli/cli_test.go`

### Delivery targets
- surface:new-spec-flat-default module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/specs_cmd_test.go
- modified: pose-mcp/internal/cli/cli_test.go

### Increments
- [x] Increment 1: Make modern flat format the default in `cmdNewSpec` (R1, R2, R3).
- [x] Increment 2: Update help catalog usage and descriptions (R5).
- [x] Increment 3: Add unit tests verifying flat layout and `spec-format status` conformance (R4).

---

## 5. Validation

### Automated
- `go test ./internal/cli -run NewSpec -count=1`
- `pose check --strict`
- `pose validate --strict`

### Requirement trace
- R1 [satisfied] surface:new-spec-flat-default check:delivery-integration test:TestNewSpec_DatePrefixScaffold evidence:integration
- R2 [satisfied] surface:new-spec-flat-default check:delivery-integration test:TestNewSpec_DatePrefixScaffold evidence:integration
- R3 [satisfied] surface:new-spec-flat-default check:delivery-integration test:TestNewSpec_DatePrefixScaffold evidence:integration
- R4 [satisfied] surface:new-spec-flat-default check:delivery-integration test:TestNewSpec_DatePrefixScaffold evidence:integration
- R5 [satisfied] surface:new-spec-flat-default check:delivery-integration test:TestHelpTopic_AllRegisteredCommands evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- surface:new-spec-flat-default -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Changed `pose new-spec` default layout to modern date-prefixed flat file `.pose/specs/YYYY-MM-DD-<slug>.md`.
- Added `--folder` flag for creating date-prefixed folder layout for specs with amends, and preserved `--legacy`.
- Updated help catalog in English and Portuguese.
- Verified format conformance and added regression tests.

### Follow-ups
- [done] All requirements delivered and verified.
