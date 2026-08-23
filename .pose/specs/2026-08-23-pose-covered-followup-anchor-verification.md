---
slug: pose-covered-followup-anchor-verification
status: in-progress
created_at: 2026-08-23
supersedes:
depends_on: pose-review-bundle-scope-isolated-listing
priority: 30
components: cli, pose-mcp
delivers: contract:covered-followup-anchor-verification
---

# Spec: Covered Follow-Up Anchor Verification and Diagnostic

## 1. Intent

### Goal
Prevent false guarantees when marking spec follow-ups with disposition `[covered: <target-slug>]`:
1. Verify whether the target spec contains a verifiable anchor (e.g. `depends_on` containing the source slug, or explicit textual reference to the source slug).
2. Emit an actionable non-blocking warning when `[covered: <target-slug>]` has no verifiable anchor in the target spec.
3. Enhance `collectFollowups` to collect follow-ups from flat spec files (`.pose/specs/*.md`) in addition to nested spec directories (`.pose/specs/*/spec.md`).

### Business value
Ensures work deferred via `[covered: <slug>]` is actively tracked or anchored in the target spec rather than disappearing silently from the active backlog.

### Constraints
- Non-blocking warning (does not fail CI for legacy or external specs without anchors, but provides clear feedback to developers and agents).
- Backward compatible across single-module and multi-module spec layouts.

---

## 2. Requirements

### Functional
- R1: `lintOneSpec` shall inspect `[covered: <target-slug>]` follow-up items and emit a `[WARNING]` if the target spec contains neither `depends_on: <source-slug>` nor a textual mention of `<source-slug>`.
- R2: `siblingSpecContent` shall locate and read spec content across nested and flat dated spec files.
- R3: `collectFollowups` shall discover follow-ups in both nested (`.pose/specs/*/spec.md`) and flat (`.pose/specs/*.md`) spec files.

### Non-functional
- Automated tests in `internal/cli` covering warning emission and flat spec follow-up discovery.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/lintspec.go`
- `pose-mcp/internal/cli/followups.go`
- `pose-mcp/internal/cli/trace_lint_test.go`
- `pose-mcp/internal/cli/followups_owner_test.go`

### Delivery targets
- contract:covered-followup-anchor-verification module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/followups.go
- modified: pose-mcp/internal/cli/trace_lint_test.go
- modified: pose-mcp/internal/cli/followups_owner_test.go

### Increments
- [x] Increment 1: Implement `siblingSpecContent` and anchor check for `[covered: <slug>]` in `lintspec.go` (R1, R2).
- [x] Increment 2: Update `collectFollowups` in `followups.go` for flat spec discovery (R3).
- [x] Increment 3: Add unit tests in `trace_lint_test.go` and `followups_owner_test.go`.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -v -run "TestFollowupsCollectsFlatSpecFiles|TestLintSpecCoveredDispositionWarns"`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:covered-followup-anchor-verification check:lint test:TestLintSpecCoveredDispositionWarnsWhenNoAnchorInTargetSpec evidence:integration
- R2 [satisfied] contract:covered-followup-anchor-verification check:lint test:TestLintSpecCoveredDispositionWarnsWhenNoAnchorInTargetSpec evidence:integration
- R3 [satisfied] contract:covered-followup-anchor-verification check:followups test:TestFollowupsCollectsFlatSpecFiles evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:covered-followup-anchor-verification -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Spec follow-up linting detects unanchored `[covered: <target-slug>]` dispositions and warns authors to include an explicit anchor.
- Follow-up collection seamlessly indexes flat dated spec layouts.

### Follow-ups
- [done] All requirements verified and delivered.
