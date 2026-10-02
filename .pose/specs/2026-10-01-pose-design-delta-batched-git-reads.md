---
slug: pose-design-delta-batched-git-reads
status: done
created_at: 2026-10-01
completed_at: 2026-10-01
delivers: capability:structural-git-batch
components: pose-mcp
task_type: feature
priority: 3
---

# Spec: Batched structural-assessment Git reads

## 1. Intent

Replace repeated Git existence/content processes inside one structural assessment
with one bounded, operation-scoped cat-file batch process.

## 2. Requirements

- R1: All successful blob reads in one assessment shall reuse one batch process.
- R2: Returned bytes and structural findings shall preserve ordinary object,
  empty-file, missing-file and filename-with-spaces behavior.
- R3: Oversized objects and unsafe request delimiters shall be rejected before
  allocating their bodies; process teardown shall occur on failure and close.
- R4: Measure the process and timing reduction using the same real Git fixture.

## 3. Technical Plan

### Artifacts

- modified: pose-mcp/internal/pose/design_delta.go
- created: pose-mcp/internal/pose/git_batch.go
- created: pose-mcp/internal/pose/git_batch_test.go
- created: .pose/specs/2026-10-01-pose-design-delta-batched-git-reads.md
- created: .pose/adr/2026-10-01-scoped-git-batch-reader-for-structural-assessment.md
- created: .pose/changelogs/unreleased/pose-design-delta-batched-git-reads.md

### Delivery targets

- capability:structural-git-batch module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

These targets identify local engine behavior exercised by this scope. Test-only
changes strengthen its regression coverage; they do not introduce a new runtime
endpoint or claim composition in Harne8.

## 4. Tasks

- [x] Add bounded reader with explicit process ownership and cancellation.
- [x] Use it only within structural-assessment collection.
- [x] Test framing, missing/empty blobs, size limits and cleanup.
- [x] Compare real Git reads and benchmark batch against independent processes.

## 5. Decisions

See ADR scoped-git-batch-reader-for-structural-assessment. Keep operation scope,
not a global pool, to avoid concurrent stream consumers or stale root binding.
No persistent cache or public command/schema changes.

## 6. Validation

Before implementation: assert byte-for-byte equality against git show; verify one
PID survives multiple requests and exits after Close; reject newline injection,
negative limits and oversized blobs. Run existing structural-delta tests and
bounded-read scenarios, then a fixed-count benchmark with real Git objects.
Performance evidence is local and does not predict every repository's timing.

### Execution log

- PR #129 CodeQL finding #122: parse blob sizes directly as native int and
  reject native integer overflow before allocation; cover native maximum,
  overflow, negative/malformed sizes, caller caps and empty blobs.
- Strict module validation passed 45/45 on 2026-10-01; full canonical validation
  is regenerated before sealing the governed review.

- Real Git framing, missing/empty/space-name parity, delimiter injection, byte
  limits and process teardown tests passed. Existing structural-delta tests passed.
- Fixed 128-read benchmark, one local sample: independent existence/content
  processes 237.107 ms versus batch 9.805 ms. Successful batch requests kept one
  process PID. This is not a benchmark of the whole check command.

### Requirement trace

- R1 [satisfied] capability:structural-git-batch evidence:integration check:delivery-integration test:TestGitBatchReadsMatchGitShowAndReuseProcess
- R2 [satisfied] test:TestGitBatchReadsMatchGitShowAndReuseProcess
- R3 [satisfied] test:TestGitBatchRejectsLimitsAndRequestInjection
- R4 [satisfied] test:BenchmarkGitBlobReadProcessVsBatch

## 7. Final Report

Implementation validated and closed through the governed review gate on 2026-10-01. Canonical strict validation passed 48/48 checks at f41b5aa; review bundle `rvb-8629f7c6a90b26ec` was fresh and approved before `pose close`.
