---
slug: pose-review-evidence-and-fixture-lifetimes
status: in-progress
created_at: 2026-11-02
completed_at:
depends_on: pose-roadmap-reviews-findings
priority: 2
components: pose-mcp
task_type: bugfix
changelog: none
---

# Spec: Review evidence and fixture lifetimes

## 1. Intent

### Goal

Make the two remaining test follow-ups of `pose-roadmap-reviews-findings` measure the inventory evidence and wait for the detached fixture they start.

### Business value

F4's test accepts a report with invented counts because it only searches for three phrases. F5 releases a detached agent in a defer and leaves the supervisor writing while temporary-directory cleanup begins. The historical cleanup race was reported under load; its exact ENOTEMPTY timing has not been reproduced here.

### Constraints

Retain the captured revisions and distinguish the later independent reproduction from the original measurement. Harne8 is private: preserve its full raw inventory in an ignored local evidence directory and version only its already-published class counts, completeness and write/coverage summaries. Do not change runtime or ownership guards.

### Non-goals

Migration apply, release tagging, Harne8 adoption, production API changes and a new executor lifecycle.

## 2. Requirements

### Functional

- R1: The cleanup report test shall compare every table class and both counts to retained inventory captures, refusing missing or duplicate classes, inconsistent counts, incomplete coverage and declared writes.
- R2: Captured evidence shall retain its actual revision and raw digest, identify pose-dist f931afe0 as a reproduction of the fcd11da9 measurement, preserve the original raw inventories locally, and exclude private Harne8 detail paths from versioned exports.
- R3: The synthetic migration dry-run test shall verify complete coverage and unchanged path/content bytes before and after the CLI call.
- R4: The wait-acknowledgement fixture shall release its agent and observe bounded terminal completion before temporary-directory cleanup.

### Compatibility

Production code, the migration dry-run contract and the private ownership guard remain unchanged. Existing historical report counts and original measurement labels remain intact.

## 3. Technical Plan

### Affected areas

Test-only inventory evidence and detached fixture cleanup in pose-mcp, plus the existing cleanup report's provenance note.

### Artifacts

- created: .pose/specs/2026-11-02-pose-review-evidence-and-fixture-lifetimes.md
- created: .pose/starts/pose-review-evidence-and-fixture-lifetimes.json
- modified: .gitignore
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-11-02-pose-roadmap-reviews-findings.md
- modified: .pose/reports/2026-11-02-pose-v7-legacy-cleanup-both-corpora.md
- modified: pose-mcp/internal/pose/roadmap_reviews_findings_test.go
- created: pose-mcp/internal/pose/cleanup_corpora_test.go
- created: pose-mcp/internal/pose/testdata/cleanup-corpora/README.md
- created: pose-mcp/internal/pose/testdata/cleanup-corpora/provenance.json
- created: pose-mcp/internal/pose/testdata/cleanup-corpora/pose-dist.json
- created: pose-mcp/internal/pose/testdata/cleanup-corpora/harne8-summary.json
- modified: pose-mcp/internal/cli/migrate_dryrun_test.go
- modified: pose-mcp/internal/cli/delegate_wait_test.go

### Technical risks

A captured inventory proves its own revision, not current corpus counts. A raw digest is an integrity reference rather than an authenticated receipt. Waiting on this fixture's terminal marker covers its writes because it has no terminal callback or auto-conclusion action. Revert this spec's test changes if these fixture assumptions cease to hold.

## 4. Tasks

- [ ] Retain and document the two captures safely
- [ ] Replace the prose-only report assertion and exercise negative cases
- [ ] Strengthen the existing synthetic read-only proof
- [ ] Join the detached fixture before cleanup
- [ ] Validate, independently review and close

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: commit both full captured inventories
- Selected option: commit the public inventory and a counts-only private-corpus export, retaining both raw captures in an ignored local directory
- Rationale: the full Harne8 capture contains private folder and transfer details that the public report never published
- Consequences: public tests can reproduce the count comparison; the raw private capture is available only in the local evidence retention directory
- Falsifier: private detail paths appear in a tracked fixture or the original captured bytes no longer match their recorded raw digest

## 6. Validation

### Strategy

Use the existing migration CLI fixture and its actual before/after tree, replace the historical report's string search with class/count validation, and exercise malformed capture/report inputs. Use guard mutations and repeated wait-acknowledgement runs under load. Run the registered pose-mcp matrix, independent Codex review, strict structure and native closeout gates. No claim is made that the exact historical cleanup race has been reproduced. The prior outcome-review handoff was consulted as knowledge:roadmap-v7-independent-outcomes-20261102; it distinguishes supplied full-suite evidence from independent focused execution.

### Requirement trace

Bind each requirement to the corresponding integration check and test after implementation.

## 7. Final Report

### Delivered scope

Implementation and validation are pending. This spec tracks the remaining F4 and F5 rather than claiming the earlier roadmap report proves them.

### Residual risks

The retained captures are historical reproductions, not measurements of today's corpora. Private evidence is deliberately not included in the public repository.

### Follow-ups

No new follow-up has been triaged in this draft.
