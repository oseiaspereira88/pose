---
slug: pose-removal-witness-freshness
status: in-progress
created_at: 2026-10-31
supersedes:
depends_on: pose-review-subject-later-removal
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:removal-witness-freshness
---

# Spec: A removal witness holds only against current history

## 1. Intent

### Goal

A review subject reads a later removal only while Git still shows that removal, and a generation says so when the recorded change sets behind it changed.

### Business value

A review subject path that a later spec removed reads as the removal the indexed delivery graph accepted. Generation bindings omit Git and the recorded change sets, so a stale graph keeps giving an accepted witness after a later touch and deletion, which live history would refuse (SR-R1-06, LR-R1-10). The milestone seals are about to hash these witnesses.

### Constraints

- The index does not record `HEAD`: it would change on every commit and stale itself.
- Reading a witness adds at most two Git calls per witness read.
- The witness text, and so every digest it enters, does not change.

### Non-goals

- Recomputing the removal rule at review time: the indexed graph stays its only computation.
- History readers outside removal witnesses, which are F1 of `pose-delivery-later-removal`.
- Budgeting removal reads at index time, which is F3 of `pose-delivery-later-removal`.

## 2. Requirements

### Functional

- R1: A removal witness shall count for a review subject only when its deleting commit is an ancestor of `HEAD` in raw history and no commit after it on `HEAD` touches the path.
- R2: A witness that fails R1, or whose check cannot be answered, shall not count, and the refusal shall name the path, the commit and why, and ask for `pose index`, instead of saying the path cannot be read.
- R3: A generation shall bind the recorded change sets of `.pose/reports/history`, so that facts report stale when one changes, and a report line without a change set changes nothing.

### Non-functional

- History is read raw, as the removal was computed, with no replace ref or graft.

### Security

- A later touch of a removed path cannot hide behind a stale index.

### Compatibility

- A generation published before this binding reads as unbound, so facts are `unknown` until the next `pose index`. The changelog says so.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`, `internal/pose/index_generation.go`, the manuals.

### Artifacts

- created: .pose/specs/2026-10-31-pose-removal-witness-freshness.md
- created: .pose/starts/pose-removal-witness-freshness.json
- created: .pose/changelogs/unreleased/pose-removal-witness-freshness.md
- modified: .pose/specs/2026-10-30-pose-review-subject-later-removal.md
- modified: .pose/specs/2026-10-30-pose-delivery-later-removal.md
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/index_generation.go
- modified: pose-mcp/internal/pose/review_subject_later_removal_test.go
- created: pose-mcp/internal/pose/removal_witness_freshness_test.go

### Delivery targets

- capability:removal-witness-freshness module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

A new generation input, `recorded-change-sets`.

## 4. Tasks

- [x] Check a witness against current history
- [x] Bind the recorded change sets
- [ ] Independent review and closeout

## 5. Decisions

- D1: The witness carries what it needs: its deleting commit. Asking Git about that commit and path is exact, cheap and needs no new index field, unlike binding `HEAD`, which would change on every commit.
- D2: Only lines with a change set are bound, so unrelated report writes do not stale every fact.
- D3: A later touch refuses even if a new index would accept the removal again: the witness then names a commit the graph no longer stands on, and reindexing settles it.

## 6. Validation

### Strategy

Real Git fixtures: a touch after the deletion, on `HEAD` and through a merge, a commit off `HEAD`, a rewritten history, an unknown commit; a replace ref that hides a touch; binding tests for a change-set line and an unrelated line; the eighteen milestones still prepare.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups

None yet.
