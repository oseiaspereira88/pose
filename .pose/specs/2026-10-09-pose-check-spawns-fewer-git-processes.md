---
slug: pose-check-spawns-fewer-git-processes
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
supersedes:
depends_on: check-worker-count-is-the-machines
priority: 1
components: pose-mcp
task_type: refactor
surface: minimal
changelog: none
delivers: capability:check-spawns-fewer-git-processes
---

# Spec: pose check spawns fewer Git processes

## 1. Intent

### Goal

Cut the Git processes `pose check --strict` spawns, without changing what it reports.

### Business value

Origin: the open follow-up of `check-worker-count-is-the-machines` (crit medium), prioritized by the maintainer on 2026-10-09. It counted 7,248 Git spawns per run as the last lever that does not depend on cores. Measured again before this change on pose-dist: 29,500 spawns and 125 s; 12,339 were one `git diff-tree` per (commit, path) pair and 9,211 one `git status` per subject path.

### Constraints

The working tree changes between operations, so its status is never cached across them; only what a commit changed, which is immutable, is cached per process.

### Non-goals

The remaining per-trailer-commit reads during indexing (`show --binary`, `diff-tree` per commit), each 1,517 per run.

## 2. Requirements

### Functional

- R1: Whether a commit changed a path shall be read once per commit and process, with the same answer as one `git diff-tree --root -r <commit> -- <path>` per pair.
- R2: Whether a subject path is dirty shall be read from one `git status` of the tree per subject, with the same answer as one `git status -- <path>` per path, and with no Git metadata nothing is dirty.
- R3: `pose check --strict` shall report exactly what it reported before.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-check-spawns-fewer-git-processes.md
- created: pose-mcp/internal/pose/git_spawn_cache.go
- created: pose-mcp/internal/pose/git_spawn_cache_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-09-21-check-worker-count-is-the-machines.md

### Delivery targets

- capability:check-spawns-fewer-git-processes module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: the follow-up proposed `cat-file --batch`; measurement showed the processes were not object reads but per-pair `diff-tree` and per-path `status`.
- Options considered: (a) cache commit paths per process and take one status snapshot per subject; (b) batch object reads through `cat-file --batch`.
- Decision: (a).
- Rationale: it removes the two calls that were 73% of the processes; (b) would address reads that were no longer the cost.
- Consequences: a long-lived server keeps the commit cache, bounded at 20,000 commits; the status snapshot lives for one subject.

## 6. Validation

### Strategy

`TestCheckGitSpawnsSnapshotMatchesStatusPerPath` compares the snapshot with `git status -- path` for a clean, edited, renamed and untracked path, a directory and a missing path, and a tree without Git. `TestCheckGitSpawnsCommitCacheMatchesDiffTreePerPath` compares the cache with `git diff-tree -- path` for a root and a later commit, a directory and a missing path. Measured on pose-dist with a counting `git` on PATH: 29,500 processes and 125 s before, 8,911 and 101 s after; the `check --strict` output is identical.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run CheckGitSpawns`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:check-spawns-fewer-git-processes check:check-git-spawns-integration evidence:integration test:TestCheckGitSpawnsCommitCacheMatchesDiffTreePerPath
- R2 [satisfied] capability:check-spawns-fewer-git-processes check:check-git-spawns-integration evidence:integration test:TestCheckGitSpawnsSnapshotMatchesStatusPerPath
- R3 [satisfied] capability:check-spawns-fewer-git-processes check:check-git-spawns-integration evidence:integration test:TestCheckGitSpawnsSnapshotMatchesStatusPerPath — the before/after `check --strict` outputs on pose-dist are identical

## 7. Final Report

### Delivered scope

The commit-path question reads each commit once per process, and subject paths read one status snapshot per subject: 29,500 → 8,911 Git processes and 125 s → 101 s for `pose check --strict` on pose-dist, with identical output.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Batch the per-trailer-commit `show --binary` and `diff-tree` reads of indexing (1,517 each per run on pose-dist), now the largest remaining Git cost (owner:@pose-maintainers crit:low review:2026-12-09)
