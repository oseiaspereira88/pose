---
slug: pose-dist-adopt-published-v6-0-4
status: in-progress
created_at: 2026-09-29
completed_at:
depends_on: pose-v6-0-4-release-readiness
priority: 0
components: docs
task_type: refactor
changelog: none
---

# Spec: Adopt the published POSE 6.0.4 machinery in pose-dist

## 1. Intent

### Goal
Record the published 6.0.4 as the engine this repository's own instance runs.
Its machinery manifest still said 6.0.0 after releasing 6.0.1 through 6.0.4.

### Constraints
Use the authenticated published binary. The release tags and ledgers stay
immutable. The distributed POSE.md is this repository's scaffold template and is
left as it is.

## 2. Requirements

- R1: The native update without force records engine_version 6.0.4.
- R2: A repeated update produces no further change and preserves the instance's
  policies, ledger and content.
- R3: The strict structural check passes after the adoption.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-29-pose-dist-adopt-published-v6-0-4.md
- modified: .pose/state/machinery-manifest.json

Run `pose update --no-self` twice with the published 6.0.4 binary and inspect the
real diff before declaring anything more.

## 4. Tasks

- [x] Authenticate the published artifact and binary.
- [x] Deliver native machinery and verify preservation and idempotence.
- [ ] Run strict checks, obtain review and close adoption.

## 5. Decisions

The 6.0.1 through 6.0.3 releases did not change the distributed machinery for
this instance, so a single adoption to 6.0.4 records the current engine without
intermediate stamps.

## 6. Validation

### Strategy
Authenticate before executing, run the update twice, then run the strict check.

### Execution log

2026-09-29: the linux_amd64 archive and `checksums.txt` of v6.0.4 match the
verified ledger; cosign verified both against the exact tag identity; the binary
`084f4ae5…080f` equals the independent rebuild and reports 6.0.4. `pose update
--no-self` changed only `engine_version` 6.0.0→6.0.4 and skipped POSE.md as the
scaffold template; the second run produced no change.

### Requirement trace

- R1 [pending] .pose/state/machinery-manifest.json
- R2 [pending] second update without change
- R3 [pending] check --strict

## 7. Final Report

### Scope delivered

Pending closeout.

### Follow-ups

None.
