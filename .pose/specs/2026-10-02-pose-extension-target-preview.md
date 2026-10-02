---
slug: pose-extension-target-preview
status: in-progress
created_at: 2026-10-02
completed_at:
priority: 4
components: pose-mcp
task_type: bugfix
delivers: surface:extension-target-preview
---
# Spec: Show and validate the extension installation target

## 1. Intent
### Goal
Make the destination explicit before extension installation writes.
### Business value
The target flag already exists but default installs do not show the destination; a missing target value silently selects the current repository.
### Constraints
Preserve signature verification and explicit --yes consent.
### Non-goals
Reimplementing doctor diagnostics already present, new prompts, or changing extension trust.

## 2. Requirements
### Functional
- R1: Every successful install plan and dry-run shall show its absolute target path before consent or writes.
- R2: A missing target argument or one followed by another flag shall be a usage error before package lookup; an absent target directory shall fail without creating it.
- R3: Explicit target, default target and rejected target tests shall prove no writes to an unintended repository; doctor profile/adoption diagnostics remain passing.
### Compatibility
Default target and successful --target behavior preserved; malformed arguments no longer silently fall back.
### Security
Do not create directories or fetch remote packages for invalid targets.

## 3. Technical Plan
### Artifacts
- modified: pose-mcp/internal/cli/extension.go
- modified: pose-mcp/internal/cli/extension_test.go
- created: .pose/changelogs/unreleased/pose-extension-target-preview.md
### Delivery targets
- surface:extension-target-preview module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
### API/contract changes
Add an extension.target field to the human install plan; existing exit-code meanings unchanged.
### Technical risks
Validate before catalog fetch; emit via renderer without increasing the direct-print baseline.

## 4. Tasks
### Implementation
- [ ] Validate target arguments and show the resolved destination.
### Validation
- [ ] Exercise default, explicit, dry-run and malformed targets.

## 5. Decisions
### Decision 1
- Date: 2026-10-02
- Decision: expose a stable plan field rather than another interactive prompt.
- Rationale: automation keeps using --yes, while preview and withheld consent both identify the affected repository.

## 6. Validation
### Strategy
Use local synthetic unsigned fixture packages; malformed target tests use a catalog ID that must never be fetched. Assert no target files or lock are written in dry-run or rejection.
### Deterministic checks
- Command: go -C pose-mcp test ./internal/cli -run 'TestExtensionInstall|TestDoctor.*(Profile|Adoption)' -count=1
- Scope: destination and existing diagnostics
- Expected: all positive and negative cases pass
### Execution log
Implementation pending.
### Requirement trace
Record after validation.
### Known gaps
Native package-channel runtime remains deferred.

## 7. Final Report
### Delivered scope
Pending validation.
### Follow-ups
None.
