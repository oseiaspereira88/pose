---
slug: pose-retire-unused-first-party-action-exception
status: done
created_at: 2026-10-20
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
completed_at: 2026-10-21
---

# Spec: Retire an unused expired Actions exception

## 1. Intent

### Goal
Restore the workflow security gate by removing the unused first-party tag-pinning waiver that expired on 2026-10-19.

### Constraints
Keep all workflow SHA pins and the waiver validation and tag rejection logic intact.

## 2. Requirements

### Functional
- R1: The exception registry shall contain no unused expired first-party tag-pinning waiver.
- R2: Every workflow action shall remain pinned to its existing full commit SHA; tag pins without a live waiver shall remain rejected.

## 3. Technical Plan

### Affected areas
The dependency-digest-pinning delivery explicitly left the unused waiver until expiry. The existing TestWorkflowSecurityContract now rejects it on the current date. Remove the entry and correct the historical test comment. Rollback restores the entry but also restores the failing gate; a future waiver requires separate explicit justification.

### Artifacts
- modified: .github/security-exceptions.json
- modified: pose-mcp/internal/version/workflow_security_test.go
- created: .pose/specs/2026-10-20-pose-retire-unused-first-party-action-exception.md

### API/contract changes
None.

### Technical risks
No runtime change. Existing validation continues enforcing owned expiring waivers and SHA pins.

## 6. Validation

### Strategy
Use the existing security contract as the regression check; run the complete module suite and native strict validation with B07.

### Deterministic checks
- Command: cd pose-mcp && go test ./internal/version -run TestWorkflowSecurityContract -count=1
- Expected: pass with no workflow or enforcement changes.

### Execution log
- 2026-10-21: Workflow contract, full race suite, native strict matrix (141/141), separate declared review, native verify and guarded close passed.
- 2026-10-20: Prior complete suite reproduced the expired first-party-actions-tag-pinning failure. No related security knowledge artifact was found.

### Requirement trace
- R1 [satisfied] test:pose-mcp/internal/version/workflow_security_test.go
- R2 [satisfied] test:pose-mcp/internal/version/workflow_security_test.go

## 7. Final Report

### Delivered scope
Remove the obsolete waiver and retain all enforcement paths.

### Residual risks
No new waiver is granted. Future tag pins require an explicitly owned live exception.

### Follow-ups
- [done] Remove the waiver intentionally retained by pose-dependency-digest-pinning through its expiry.
