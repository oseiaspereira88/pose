---
slug: pose-history-readers-read-raw
status: in-progress
created_at: 2026-10-31
supersedes:
depends_on: pose-delivery-later-removal
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:history-readers-read-raw
---

# Spec: Review reads the same history as delivery, from its first commit

## 1. Intent

### Goal

Every history reader of structural review reads raw object history, as the delivery graph does, and a trailer change set that begins at the repository's first commit resolves.

### Business value

The delivery graph ignores replace refs and grafts, while structural review honoured them, so one commit could mean two things to two gates (LR-R4-02). A spec whose first commit is the repository root had no parent to diff from, so a new project's first spec had no change set, and a later removal of what it created was refused (LR-R4-03).

### Constraints

- A change set of a commit with a parent resolves to the same identity as before.
- Raw history is one rule in one place, not a flag at each call.

### Non-goals

- Git calls outside structural review and the delivery graph: discovery, assessment freshness, docs and release reads do not attribute history to specs.
- Budgets for removal reads, which are F3 of `pose-delivery-later-removal`.

## 2. Requirements

### Functional

- R1: `gitCommand`, and every Git read of `review_path_attribution.go`, `git_batch.go` and `design_delta.go`, shall read raw object history: no replace ref and no graft, from the environment or the repository.
- R2: A trailer change set whose first commit object has no parent shall resolve, with the empty tree as base, its paths and diff taken from that commit. A shallow boundary, whose object names a parent the clone lacks, stays unresolved.
- R3: Readers of a change set's range, the review subject, attribution and the policy baseline, shall accept a base that is the empty tree, reading it as the range up to the head.

### Non-functional

- No additional Git call for a commit whose parent resolves.

### Security

- A replace ref or graft cannot make review judge another history than delivery.

### Compatibility

- A repository with replace refs or grafts gets different review inputs: rerun `pose index` and reseal. The manual and changelog say so.

## 3. Technical Plan

### Affected areas

`internal/pose/git_snapshot_integrity.go`, `internal/pose/spec_trailers.go`, `internal/pose/review_path_attribution.go`, `internal/pose/git_batch.go`, `internal/pose/design_delta.go`, `internal/pose/review_bundle.go`, `internal/cli/artifact_integrity.go`, the manuals.

### Artifacts

- created: .pose/specs/2026-10-31-pose-history-readers-read-raw.md
- created: .pose/starts/pose-history-readers-read-raw.json
- created: .pose/changelogs/unreleased/pose-history-readers-read-raw.md
- modified: .pose/specs/2026-10-30-pose-delivery-later-removal.md
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- modified: pose-mcp/internal/pose/spec_trailers.go
- modified: pose-mcp/internal/pose/review_path_attribution.go
- modified: pose-mcp/internal/pose/git_batch.go
- modified: pose-mcp/internal/pose/design_delta.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_policy_baseline.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- created: pose-mcp/internal/pose/history_readers_raw_test.go
- created: .pose/specs/2026-10-31-pose-history-readers-read-raw.amendments.jsonl
- created: pose-mcp/internal/cli/change_set_root_test.go

### Delivery targets

- capability:history-readers-read-raw module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`resolved_base` of a change set can be the empty tree.

## 4. Tasks

- [x] One raw rule in gitCommand, used by the readers
- [x] Resolve a change set from the root
- [ ] Independent review and closeout

## 5. Decisions

- D1: `gitCommand` carries the raw rule through its environment, so every caller gets it, and `RawHistoryGit` becomes a call of it.
- D2: The empty tree is the base of a root change set: it is what Git itself diffs a root commit against, and it keeps `resolved_base` filled, which readers already require.

## 6. Validation

### Strategy

Real repositories: a replace ref and a graft that would change an attribution, a subject and a delta; a first commit carrying the trailer, through the collector, the graph, a later removal, `pose index`, review preparation and `check --strict`.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups

None yet.
