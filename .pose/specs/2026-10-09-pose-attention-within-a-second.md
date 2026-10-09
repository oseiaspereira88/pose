---
slug: pose-attention-within-a-second
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
supersedes:
depends_on: pose-agency-readiness-pilot
priority: 1
components: pose-mcp
task_type: refactor
surface: minimal
delivers: capability:attention-within-a-second
---

# Spec: Attention within about a second

## 1. Intent

### Goal

Bring `pose state --attention` close to its 1 s target on large corpora without changing what it answers.

### Business value

Origin: the open follow-up of `pose-agency-readiness-pilot` (crit medium), prioritized by the maintainer on 2026-10-09. The pilot measured 30 to 37 s on Harne8; Attention is read before every phase, so its cost is paid on every step.

### Constraints

Read-only and exact: the obligations, their ids and satisfaction must be identical; nothing is cached across operations, because specs and bundles change between them.

### Non-goals

Making bundle preparation per in-progress spec cheaper.

## 2. Requirements

### Functional

- R1: One obligation projection shall read the specs directory once, for the store and for every store its artifact resolver opens, with the same listing and no shared mutable copy.
- R2: One projection shall read each review bundle's scope once for scoped bundle listings, with the same bundles listed.
- R3: The projection shall answer exactly what it answered before.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-attention-within-a-second.md
- created: pose-mcp/internal/pose/spec_snapshot_test.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-10-04-pose-agency-readiness-pilot.md
- created: .pose/changelogs/unreleased/pose-attention-within-a-second.md

### Delivery targets

- capability:attention-within-a-second module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: profiling showed the cost was I/O, not parsing: `GetSpec` lists every spec to find one, and each producer called it per spec; scoped bundle listings decoded every bundle per call. A content-keyed parse memo was tried first and cut little, because it still read every file.
- Options considered: (a) a per-operation snapshot of the spec listing and bundle scopes; (b) a process-wide memo keyed by content.
- Decision: (a).
- Rationale: it removes the repeated reads within one answer and cannot go stale across operations.
- Consequences: a store outside a projection behaves exactly as before.

## 6. Validation

### Strategy

`TestAttentionSnapshotListsWhatTheDiskLists` compares the snapshot listing with a direct one, filtered and unfiltered, and checks that a caller's edit does not reach it and that `GetSpec` still returns the body. `TestAttentionSnapshotListsTheSameBundles` compares scoped bundle listings. Measured over three runs each, with identical obligation ids and satisfaction: Harne8 2.27 s → 1.33 s, pose-dist 1.39 s → 0.91 s.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run 'AttentionSnapshot|TestObligation|AttentionSource'`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:attention-within-a-second check:attention-snapshot-integration evidence:integration test:TestAttentionSnapshotListsWhatTheDiskLists
- R2 [satisfied] capability:attention-within-a-second check:attention-snapshot-integration evidence:integration test:TestAttentionSnapshotListsTheSameBundles
- R3 [satisfied] capability:attention-within-a-second check:attention-snapshot-integration evidence:integration test:TestAttentionSnapshotListsWhatTheDiskLists — obligation ids and satisfaction identical before and after on Harne8 and pose-dist

## 7. Final Report

### Delivered scope

A projection reads the specs once per project and each bundle's scope once: Attention takes 1.33 s on Harne8 (the pilot measured 30 to 37 s; 2.27 s before this change) and 0.91 s on pose-dist, with identical answers.

### Residual risks

- On Harne8 the closeout producer still prepares a bundle per in-progress spec, about 1.3 s of the read.
- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
