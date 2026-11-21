---
slug: pose-review-dispatch-materializes-sealed-gitlinks
status: in-progress
created_at: 2026-11-20
started_at: 2026-11-20
depends_on: pose-review-subject-submodule-classification
priority: 1
components: pose-mcp
delivers: capability:review-dispatch-materializes-sealed-gitlinks
---

# Spec: Materialize sealed gitlinks during review dispatch

## 1. Intent

### Goal

Allow a governed review to reach the reviewer when its sealed subject includes a materialized gitlink. A detached checkout of the parent commit currently leaves an empty directory at the gitlink and fails worktree identity verification.

### Business value

Harne8 can obtain native governed approval for changes that include a pose-dist pin.

### Constraints

Keep the selected origin, sealed commit, setup deadline, checkout verifiers, process containment and resource cleanup authoritative.

### Non-goals

Initialize unrelated submodules, resolve a remote URL or review changes inside a submodule.

## 2. Requirements

- R1: Materialize each scoped gitlink at the commit recorded in the sealed parent tree and bound to the subject digest. Use only committed objects already available in the selected local origin, including nested gitlinks required to verify the scoped repository.
- R2: Keep Git metadata private to the run and give equivalent repeated checkouts the same snapshot. Preserve origin files and Git metadata.
- R3: Fail before running the adapter when needed objects are unavailable or the pin, source worktree or resulting checkout fails verification. Apply the existing setup process, capture and cleanup limits.

## 3. Technical Plan

### Approach

Use the validated origin's local submodule and stage a standalone detached checkout inside the private review worktree. Keep its private Git directory adjacent to that worktree. See the [ADR](../adr/2026-11-20-materialize-review-gitlinks-from-the-sealed-local-origin.md).

### Artifacts

- created: .pose/specs/2026-11-20-pose-review-dispatch-materializes-sealed-gitlinks.md
- created: .pose/adr/2026-11-20-materialize-review-gitlinks-from-the-sealed-local-origin.md
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/review_dispatch_gitlinks.go
- modified: pose-mcp/internal/cli/git_snapshot_integrity_test.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go

### Delivery targets

- capability:review-dispatch-materializes-sealed-gitlinks module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

- [x] Materialize the sealed scoped Git repositories under the existing setup budget.
- [x] Exercise success and rejection through native dispatch.
- [ ] Qualify the candidate against the engine gates and a real Harne8 review.

## 5. Decisions

Use local shallow object transfer with private Git administration and stable relative markers. The ADR records the trust boundary and lifecycle.

## 6. Validation

Run the CLI regression against the parent first. It must reach sealing and fail before the adapter with the empty submodule identity error. After the fix, assert the exact scoped revision and nested payload, absence of unrelated checkouts, skipped hooks, unchanged origin and worktree cleanup. Exercise repeat dispatch for a stable tree digest and hidden-worktree changes for rejection before execution. Run existing dispatch, containment and snapshot tests, then CI and the native Harne8 review.

### Requirement trace

- R1 [satisfied] test:TestReviewDispatchMaterializesSealedGitlinks
- R2 [satisfied] test:TestReviewDispatchMaterializesSealedGitlinks
- R3 [satisfied] test:TestReviewDispatchMaterializesSealedGitlinks

## 7. Final Report

The CLI fixture reached the empty gitlink identity error on the parent commit and passed after local materialization. Candidate validation and a real Harne8 review remain pending.
