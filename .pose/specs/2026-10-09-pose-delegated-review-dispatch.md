---
slug: pose-delegated-review-dispatch
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-brief
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Adapter policy and examples
- [ ] Disposable worktree runner
- [ ] Run record and draft capture
- [ ] Failure classification

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

## 6. Validation

### Strategy

A fake adapter (shell script) exercises success, timeout, non-zero exit and a run that edits files; the user's working tree is byte-identical afterwards.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
