---
slug: pose-spec-transfer-completion-and-references
status: done
created_at: 2026-09-26
supersedes:
depends_on: pose-spec-transfer-reconcile-terminal
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:spec-transfer-completion-and-references
completed_at: 2026-09-27
---

# Spec: Transfer completion, `after` references and redirect stubs

## 1. Intent

### Goal
Close three gaps a rehearsal of seven sequential reconciliations exposed in
spec transfer.

### Business value
After seven reconciliations in a populated copy of Harne8, resuming the first
operation failed with `compare-and-swap-conflict` because later operations had
rewritten files it touched; `harne8-abm-platform` kept `after:
spec:pose-abm-*` refs to retired coordinators, which resolved to
`source-revision-unavailable`; and `lint-spec --all` failed seven times on the
redirect stubs the transfer itself writes.

### Constraints
Keep plan and redirect formats. A bare `after` entry names a milestone and
must never match a spec of the same name. A stub is exempt from lint only when
its redirect, plan and activation verify and the file is byte-identical to the
rendered stub.

### Non-goals
Do not change transfer phases or authorization.

## 2. Requirements

- R1: Resuming an operation that every affected project journaled as activated reports that status and writes nothing, even when later operations rewrote the files it touched.
- R2: Impact discovery and rewriting cover `after` entries that name an artifact kind; bare `after` entries are untouched.
- R3: `lint-spec` accepts a verified transfer stub and reports its canonical task; an edited stub or a stub without a verified redirect is linted as a spec.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/spec_transfer.go` (`completedSpecTransfer`,
`explicitAfterRefs`, `VerifiedTransferStub`) and `internal/cli/lintspec.go`.

### Artifacts
- created: .pose/specs/2026-09-26-pose-spec-transfer-completion-and-references.md
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/pose/spec_transfer_reconcile_test.go
- modified: pose-mcp/internal/cli/lintspec.go

### Delivery targets
- contract:spec-transfer-completion-and-references module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Resume checks activation receipts before advancing. The roadmap scan and
rewrite add explicit `after` entries. Lint asks the engine to verify a
superseded spec as a transfer stub before section checks. Rollback is a revert.

## 4. Tasks

- [x] Reproduce the three failures in the reconciliation rehearsal.
- [x] Write regressions and prove the resume and `after` ones fail without the change.
- [x] Implement the three fixes.
- [x] Run the matrix, review and close.

## 5. Decisions

Amends nothing structural: the transfer ADR already requires current
dependency links to follow the verified redirect and one executable authority.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1-R3 | matrix check `spec-authority-transfer-integration` | Completed resume reported; explicit `after` rewritten, bare kept; verified stub recognised, edited stub refused. |
| Module matrix | `pose validate --strict --module pose-mcp --report` | Required checks pass. |

### Execution log
2026-09-26: with the completion check disabled the resume regression failed
with `compare-and-swap-conflict`; without the `after` scan the roadmap stayed
unrewritten. The print-site ratchet kept `lintspec.go` at its baseline by
reporting through the renderer. `go test ./...` and `go vet ./...` pass.

At `eb143d6` the full matrix passed 29/29. Bundle `rvb-26af7996a3806689` was approved by
attestation `rva-79d4345bfa1a4657`, recorded by the agent under explicit authorization
from the user to self-attest; the superseded foundation, transfer and ABM
executor reviews were resealed and reattested.

### Requirement trace
- R1 [satisfied] test:TestSpecTransferResumeReportsACompletedOperationAfterLaterRewrites
- R2 [satisfied] test:TestSpecTransferRewritesExplicitAfterRefsAndVerifiesTheStub
- R3 [satisfied] test:TestSpecTransferRewritesExplicitAfterRefsAndVerifiesTheStub

## 7. Final Report

### Delivered scope
Completed transfers resume as a report, explicit after refs follow a transfer, and verified redirect stubs pass lint.

### Residual risks
None beyond the fail-closed behaviour recorded in the requirements.

### Follow-ups
