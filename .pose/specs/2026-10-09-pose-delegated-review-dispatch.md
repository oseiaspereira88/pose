---
slug: pose-delegated-review-dispatch
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-brief
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-review-dispatch
---

# Spec: Dispatch a delegated review to a configured adapter

## 1. Intent

### Goal

Run a configured reviewer command on a disposable copy of the sealed commit and record the run and its draft, replacing personal launcher scripts.

### Business value

The 2026-10-09 launcher had to solve per run what the engine should solve once: the reviewer's sandbox made `.git` read-only, a validation was killed for lack of memory, and the implementer chose model and flags by hand.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

POSE depends on no vendor CLI. The repository the user works in is never written by a reviewer run. Requirements follow the maintainer's decisions recorded in the accepted ADR on 2026-10-09.

## 2. Requirements

### Functional

- R1: `.pose/policy/reviewers.json` shall declare adapters: a command template, the vendor and model it runs, a timeout and an execution budget; POSE ships examples for Codex (`codex exec`) and Claude Code (`claude -p`) and enables none.
- R2: `pose review dispatch <bundle> --via <adapter>` shall create a disposable worktree at the bundle's sealed commit, pass the brief on stdin, and remove the worktree afterwards.
- R3: A run shall be recorded under `.pose/review-runs/` with adapter, vendor, model, start and end time, exit code, brief digest, transcript digest and the reviewer's draft attestation or findings.
- R4: A run that changes the sealed commit's files, exceeds its timeout or budget, or exits non-zero shall be recorded as failed, never as a review.
- R5: Dispatch alone shall not record an attestation; recording a run's conclusion is the engine's verified step, specified by `pose-delegated-review-capability`, and runs only when the run passes every check there.
- R6: With no adapter configured, dispatch shall print the brief and the configuration needed, not fail silently.
- R7: `pose review dispatch` shall accept every brief kind (`review`, `adjudication`, `smoke`) with the same run record.

## 3. Technical Plan

### Affected areas

Adapter policy reader, worktree lifecycle, process runner with timeout, run record writer, and the `review dispatch` subcommand.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-dispatch.md
- created: .pose/starts/pose-delegated-review-dispatch.json
- created: .pose/changelogs/unreleased/pose-delegated-review-dispatch.md
- created: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/cli/review_dispatch.go
- created: pose-mcp/internal/cli/review_dispatch_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/pose/review_dispatch_unix.go
- created: pose-mcp/internal/pose/review_dispatch_windows.go

### Delivery targets

- capability:delegated-review-dispatch module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

- [x] Adapter policy and examples
- [x] Disposable worktree runner
- [x] Run record and draft capture
- [x] Failure classification

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

## 6. Validation

### Strategy

A fake adapter (shell script) exercises success, timeout, non-zero exit and a run that edits files; the user's working tree is byte-identical afterwards.

### Execution log

2026-10-10, first real run: `pose review dispatch spec:pose-ci-avoids-anonymous-rate-limits --via codex --apply` with the example Codex adapter (gpt-6.1-sol, read-only) ran on a worktree at aa29027d, received the generated brief and returned a full review — a decision, a judgment per planned criterion with file and line evidence, and one medium defect in that spec. The run was recorded as failed because the worktree showed `.pose/assessments/technical-debt.md` and `.pose/state/technical-debt.json` modified: the reviewer ran the `assess tech-debt` tool the plan requires, which regenerates derived state. The change check now ignores derived state (`.pose/assessments`, `.pose/state`, `.pose/indexes`, `.pose/results`, `.pose/review-bundles`, `.pose/review-runs`, `.pose/reports/history`) and still fails a run that edits anything else; the test covers both.

### Execution log

2026-10-10, independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): medium severity. The timeout killed only the adapter's process; a child holding stdout (the shape of `codex exec` and `claude -p`) kept the dispatch waiting until it ended. The adapter now runs in its own process group, the group is killed on timeout and after the run, and `WaitDelay` releases the pipes. `TestReviewDispatchTimeoutKillsTheAdaptersChildren` took 30s on the previous code and fails; it now returns in under a second.

### Requirement trace

- R1 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRunsAndRecordsAdapters
- R2 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRunsAndRecordsAdapters
- R3 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRunsAndRecordsAdapters
- R4 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRunsAndRecordsAdapters test:TestReviewDispatchTimeoutKillsTheAdaptersChildren
- R5 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRunsAndRecordsAdapters
- R6 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRunsAndRecordsAdapters
- R7 [satisfied] capability:delegated-review-dispatch evidence:integration test:TestReviewDispatchRecordsEveryBriefKind

## 7. Final Report

### Delivered scope

`pose review dispatch` runs a configured reviewer adapter on a disposable worktree at the sealed commit with the brief on stdin, records completed and failed runs with their transcripts under `.pose/review-runs/`, and records no attestation.

### Residual risks

- The worktree is disposable but the reviewer process runs with the user's permissions: an adapter can still write outside the worktree. The example adapters run read-only (`codex exec -s read-only`, `claude --permission-mode plan`).
- Runs are append-only by convention here; immutability and the rerun rule are spec `pose-delegated-review-attempt-ledger`.

### Follow-ups
