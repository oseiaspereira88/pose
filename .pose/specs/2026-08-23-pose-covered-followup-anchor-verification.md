---
slug: pose-covered-followup-anchor-verification
status: done
created_at: 2026-08-23
supersedes:
depends_on: pose-review-bundle-scope-isolated-listing
priority: 30
components: cli, pose-mcp
delivers: contract:covered-followup-anchor-verification
completed_at: 2026-08-23
---

# Spec: Covered Follow-Up Anchor Verification and Diagnostic

## 1. Intent

### Goal
Prevent false guarantees when marking spec follow-ups with disposition `[covered: <target-slug>]`:
1. Verify whether the target spec contains a verifiable anchor (e.g. `depends_on` containing the source slug, or explicit textual reference to the source slug).
2. Emit an actionable non-blocking warning when `[covered: <target-slug>]` has no verifiable anchor in the target spec.
3. Enhance `collectFollowups` to collect follow-ups from flat spec files (`.pose/specs/*.md`) in addition to nested spec directories (`.pose/specs/*/spec.md`).
4. Establish explicit status lifecycle (`staged` -> `submitted` / `dismissed`) for POSE contributor and feedback artifacts, and mandate explicit user adjudication before staging or submitting upstream.

### Business value
Ensures work deferred via `[covered: <slug>]` is actively tracked or anchored in the target spec rather than disappearing silently from the active backlog. Gives developers and agents full visibility and control over contributor feedback artifacts.

### Constraints
- Non-blocking warning (does not fail CI for legacy or external specs without anchors, but provides clear feedback to developers and agents).
- Backward compatible across single-module and multi-module spec layouts.

---

## 2. Requirements

### Functional
- R1: `lintOneSpec` shall inspect `[covered: <target-slug>]` follow-up items and emit a `[WARNING]` if the target spec contains neither `depends_on: <source-slug>` nor a textual mention of `<source-slug>`.
- R2: `siblingSpecContent` shall locate and read spec content across nested and flat dated spec files.
- R3: `collectFollowups` shall discover follow-ups in both nested (`.pose/specs/*/spec.md`) and flat (`.pose/specs/*.md`) spec files.
- R4: `pose contribute` shall support status lifecycle commands (`list --status`, `mark-submitted`, `submit`, `dismiss`) and track metadata (`status`, `submitted_at`, `upstream_issue`).
- R5: Contributor documentation in `AGENTS.md` and `POSE.md` shall mandate explicit user adjudication for both staging and submitting feedback.

### Non-functional
- Automated tests in `internal/cli` covering warning emission, flat spec follow-up discovery, and contributor lifecycle transitions.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/lintspec.go`
- `pose-mcp/internal/cli/followups.go`
- `pose-mcp/internal/cli/contribute.go`
- `pose-mcp/internal/cli/report_limitation.go`
- `pose-mcp/internal/cli/trace_lint_test.go`
- `pose-mcp/internal/cli/followups_owner_test.go`
- `pose-mcp/internal/cli/contribute_test.go`
- `AGENTS.md`
- `POSE.md`

### Delivery targets
- contract:covered-followup-anchor-verification module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/followups.go
- modified: pose-mcp/internal/cli/contribute.go
- modified: pose-mcp/internal/cli/report_limitation.go
- modified: pose-mcp/internal/cli/trace_lint_test.go
- modified: pose-mcp/internal/cli/followups_owner_test.go
- modified: pose-mcp/internal/cli/contribute_test.go
- modified: AGENTS.md
- modified: POSE.md

### Increments
- [x] Increment 1: Implement `siblingSpecContent` and anchor check for `[covered: <slug>]` in `lintspec.go` (R1, R2).
- [x] Increment 2: Update `collectFollowups` in `followups.go` for flat spec discovery (R3).
- [x] Increment 3: Add contributor status lifecycle and user adjudication in `contribute.go`, `report_limitation.go`, `AGENTS.md`, and `POSE.md` (R4, R5).
- [x] Increment 4: Add unit tests in `trace_lint_test.go`, `followups_owner_test.go`, and `contribute_test.go`.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -v -run "TestFollowupsCollectsFlatSpecFiles|TestLintSpecCoveredDispositionWarns|TestContribute"`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:covered-followup-anchor-verification check:lint test:TestLintSpecCoveredDispositionWarnsWhenNoAnchorInTargetSpec evidence:integration
- R2 [satisfied] contract:covered-followup-anchor-verification check:lint test:TestLintSpecCoveredDispositionWarnsWhenNoAnchorInTargetSpec evidence:integration
- R3 [satisfied] contract:covered-followup-anchor-verification check:followups test:TestFollowupsCollectsFlatSpecFiles evidence:integration
- R4 [satisfied] contract:covered-followup-anchor-verification check:contribute test:TestContributeStatusLifecycleTransitions evidence:integration
- R5 [satisfied] contract:covered-followup-anchor-verification check:contribute test:TestContributeLifecycleCommands evidence:integration

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
- POSE Contributor Mode tracks contribution artifact lifecycle (`staged` -> `submitted` / `dismissed`) and mandates explicit user confirmation before staging or submitting feedback.

### Follow-ups
- [done] All requirements verified and delivered.
