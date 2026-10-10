---
slug: pose-review-subject-classifies-engine-records
status: done
created_at: 2026-10-10
completed_at: 2026-10-10
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
- R2: `.pose/review-runs/`, `.pose/events/`, `.pose/investigations/`, `.pose/telemetry.json`, `.pose/continuous-closeout.json` and `.pose/pose-validate.log` shall be classified as derived evidence, `.pose/schema-version` and `.pose/usage/` as governance, and `.pose/LICENSE` and `.pose/NOTICE` as documentation, like the repository's own.
- R3: A test shall read the names the engine writes under `.pose/` from every package of its source, in any case and with any extension, and a second test shall check every file a fresh install and an update leave under `.pose/`; each fails when a name has no class.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-review-subject-classifies-engine-records.md
- created: .pose/starts/pose-review-subject-classifies-engine-records.json
- created: .pose/changelogs/unreleased/pose-review-subject-classifies-engine-records.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/cli/install_review_subject_test.go
- created: pose-mcp/internal/pose/review_bundle_engine_dirs_test.go
- created: .pose/specs/2026-10-10-pose-review-subject-classifies-engine-records.amendments.jsonl

### Delivery targets

- governance:review-subject-classifies-engine-records module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Basis: R1, R2, R3
- Context: the classifier is a hand-kept list; the ledger directory was added by `pose-signed-legacy-attestation-ledger` and never added to it.
- Decision: classify the five names, and guard the list with a test that scans the engine's source for `.pose/<name>` literals and `filepath.Join(…, ".pose", "<name>")` calls.
- Rationale: a review of the classifier alone would not have found the gap; the scan found it, and four more.

## 6. Validation

### Execution log

2026-10-10, independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): high and medium. The scan read only three packages, so `.pose/usage/` (the usage verdict journal, written by `internal/usage`) had no class and the seal refused it; and the patterns skipped names with an extension, missing `.pose/telemetry.json`, `.pose/continuous-closeout.json` and `.pose/pose-validate.log`. The scan now walks the whole module and matches extensions; the four names are classified, R2 and R3 amended. The widened scan fails on the previous classifier listing exactly those four.

2026-10-10, second independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): high severity. `.pose/LICENSE` and `.pose/NOTICE`, which install copies, had no class, and the scan's patterns skipped upper-case names. They are classified as documentation; the patterns accept any case and underscores (with `AGENTS.md` and `POSE.md`, which only follow `.pose` in a `git status` argument list, excluded); and because the scan reads string patterns, a second test checks every file a fresh install and an update actually write under `.pose/`. Both tests fail on the previous classifier naming exactly these two files.

2026-10-10, third independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): the code was approved; the spec failed `lint-spec --strict` once closed. Its amendment log began with a `semantic` event and no baseline, so R1 and D1 read as added after the start, and D1 had no `Basis`. D1 now names its basis and a baseline event acknowledges the current nodes; `lint-spec --strict` on a copy marked done passes. The amendment gate runs only on done specs, which is why it surfaced after `pose close`.

### Strategy

`TestEveryEngineDirectoryUnderPoseIsClassified` and `TestLegacyLedgerIsAGovernanceRecord` fail on the previous classifier (the scan lists events, investigations, review-ledgers, review-runs and schema-version) and pass on the new one.

### Requirement trace

- R1 [satisfied] governance:review-subject-classifies-engine-records evidence:unit test:TestLegacyLedgerIsAGovernanceRecord
- R2 [satisfied] governance:review-subject-classifies-engine-records evidence:unit test:TestEveryEngineDirectoryUnderPoseIsClassified test:TestUsageVerdictJournalIsAGovernanceRecord
- R3 [satisfied] governance:review-subject-classifies-engine-records evidence:unit test:TestEveryEngineDirectoryUnderPoseIsClassified test:TestEveryFileTheEngineWritesUnderPoseIsClassified

## 7. Final Report

### Delivered scope

Every directory and file the engine writes under `.pose/` has a review-subject class; the legacy ledger is reviewed as governance.

### Residual risks

- The scan reads string literals; a path built another way escapes it, unless install or update writes it, which the second test checks. A path only other commands write is covered by the scan alone.

### Follow-ups
