---
slug: pose-review-subject-classifies-engine-records
status: in-progress
created_at: 2026-10-10
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: governance:review-subject-classifies-engine-records
---

# Spec: The review subject classifies every record the engine writes

## 1. Intent

### Goal

Every path the engine writes under `.pose/` shall have a review-subject class, so a scope that carries one can be sealed.

### Business value

On 2026-10-10 Harne8's `harne8-adopt-pose-v7-2-0` could not be sealed: `pose review bundle --seal` refused `unclassified review subject path .pose/review-ledgers/legacy-harne8-agents-20261010T032511Z.json`, a ledger `pose adopt signed-attestations` itself wrote in 7.3.0. Any project that adopted signed attestations through the ledger is stuck the same way. A scan of the engine's source found four more names it writes and the classifier did not know.

### Constraints

A path that decides authority is reviewed with its scope; evidence about work is not.

## 2. Requirements

### Functional

- R1: `.pose/review-ledgers/` shall be classified as governance and included in the review subject.
- R2: `.pose/review-runs/`, `.pose/events/` and `.pose/investigations/` shall be classified as derived evidence, and `.pose/schema-version` as governance.
- R3: A test shall read the directories the engine writes under `.pose/` from its own source and fail when one has no class.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-review-subject-classifies-engine-records.md
- created: .pose/starts/pose-review-subject-classifies-engine-records.json
- created: .pose/changelogs/unreleased/pose-review-subject-classifies-engine-records.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_bundle_engine_dirs_test.go

### Delivery targets

- governance:review-subject-classifies-engine-records module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: the classifier is a hand-kept list; the ledger directory was added by `pose-signed-legacy-attestation-ledger` and never added to it.
- Decision: classify the five names, and guard the list with a test that scans the engine's source for `.pose/<name>` literals and `filepath.Join(…, ".pose", "<name>")` calls.
- Rationale: a review of the classifier alone would not have found the gap; the scan found it, and four more.

## 6. Validation

### Strategy

`TestEveryEngineDirectoryUnderPoseIsClassified` and `TestLegacyLedgerIsAGovernanceRecord` fail on the previous classifier (the scan lists events, investigations, review-ledgers, review-runs and schema-version) and pass on the new one.

### Requirement trace

- R1 [satisfied] governance:review-subject-classifies-engine-records evidence:unit test:TestLegacyLedgerIsAGovernanceRecord
- R2 [satisfied] governance:review-subject-classifies-engine-records evidence:unit test:TestEveryEngineDirectoryUnderPoseIsClassified
- R3 [satisfied] governance:review-subject-classifies-engine-records evidence:unit test:TestEveryEngineDirectoryUnderPoseIsClassified

## 7. Final Report

### Delivered scope

Every directory and file the engine writes under `.pose/` has a review-subject class; the legacy ledger is reviewed as governance.

### Residual risks

- The scan reads string literals; a path built another way escapes it.

### Follow-ups
