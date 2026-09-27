---
slug: pose-review-bundle-classifies-transfer-records
status: done
created_at: 2026-09-26
supersedes:
depends_on: pose-spec-transfer-reconcile-terminal
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:review-bundle-classifies-transfer-records
completed_at: 2026-09-27
---

# Spec: Review bundles classify transfer records

## 1. Intent

### Goal
Let a review bundle seal a change that records a spec transfer or
reconciliation under `.pose/transfers/`.

### Business value
After Harne8 retired its ABM coordinators, the review bundle of
`harne8-abm-spec-authority-reconciliation` refused with 43 `unclassified
review subject path .pose/transfers/...` blockers: the classifier refuses
unknown paths by design, and the transfer journal, archived sources and
redirects were never listed.

### Constraints
Transfer records are authority records and must be reviewed as governance.
The transfer lock is runtime state and must not be.

### Non-goals
Do not change other classifications.

## 2. Requirements

- R1: Plans, receipts, archived sources and redirects under `.pose/transfers/` classify as governance and are included in the review subject.
- R2: `.pose/transfers/.authority-transfer.lock` is never classified as governance.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/review_bundle.go` (`reviewBundlePathClass`).

### Artifacts
- created: .pose/specs/2026-09-26-pose-review-bundle-classifies-transfer-records.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/spec_transfer_reconcile_test.go

### Delivery targets
- contract:review-bundle-classifies-transfer-records module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Add `.pose/transfers/` to the governance prefixes and exempt the lock.
Rollback is a revert.

## 4. Tasks

- [x] Reproduce the refusal on the Harne8 reconciliation bundle.
- [x] Write the regression and prove it fails without the change.
- [x] Classify the records.
- [x] Run the matrix, review and close.

## 5. Decisions

Transfer records decide who owns a spec, so they are reviewed like roadmaps
and policy rather than skipped as derived evidence.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1/R2 | matrix check `spec-authority-transfer-integration` | Records classify as governance; lock does not. |
| Harne8 consumer | `review bundle spec:harne8-abm-spec-authority-reconciliation` | No unclassified path blockers. |

### Execution log
2026-09-26: the Harne8 reconciliation bundle reported 43 unclassified
`.pose/transfers/` paths with the pinned engine and none with a candidate
carrying this change. Without the prefix the regression reported four
misclassified records. `go test ./...` and `go vet ./...` pass.

At `f6e6215` the full matrix passed 29/29. Bundle `rvb-8bf4dc1da432f891` was
approved by attestation `rva-b1b39bf693820490`, recorded by the agent under
explicit authorization from the user to self-attest; superseded reviews were
resealed and reattested. The lock file committed with the reconciliation
journal was removed from version control and ignored (`df16997`).

### Requirement trace
- R1 [satisfied] test:TestSpecTransferJournalClassifiesAsGovernance
- R2 [satisfied] test:TestSpecTransferJournalClassifiesAsGovernance

## 7. Final Report

### Delivered scope
Review bundles seal changes that record spec transfers and reconciliations.

### Residual risks
None identified.

### Follow-ups
