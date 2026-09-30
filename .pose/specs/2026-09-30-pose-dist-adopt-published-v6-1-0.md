---
slug: pose-dist-adopt-published-v6-1-0
status: in-progress
created_at: 2026-09-30
completed_at:
depends_on: pose-v6-1-0-release-readiness
priority: 0
components: docs
task_type: refactor
changelog: none
---

# Spec: Adopt the published POSE 6.1.0 machinery in pose-dist

## 1. Intent

### Goal
Record the published 6.1.0 as the engine this repository's own instance runs.

### Constraints
Use the authenticated published binary. The release tags and ledgers stay
immutable. The distributed POSE.md is this repository's scaffold template and is
left as it is.

## 2. Requirements

- R1: The native update without force records engine_version 6.1.0.
- R2: A repeated update produces no further change and preserves the instance's
  policies, ledger and content.
- R3: The strict structural check passes after the adoption.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-30-pose-dist-adopt-published-v6-1-0.md
- modified: .pose/state/machinery-manifest.json

Run `pose update --no-self` twice with the published 6.1.0 binary and inspect the
real diff before declaring anything more.

## 4. Tasks

- [x] Authenticate the published artifact and binary.
- [x] Deliver native machinery and verify preservation and idempotence.
- [ ] Run strict checks, obtain review and close adoption.

## 5. Decisions

The machinery 6.1.0 ships for this instance is this repository's own source, so
the update records digests and the engine version without rewriting content.

## 6. Validation

### Strategy
Authenticate before executing, run the update twice, then run the strict check.

### Execution log

2026-09-30: the linux_amd64 archive, `checksums.txt` and its Sigstore bundle of
v6.1.0 match the verified ledger; cosign verified `checksums.txt` against the
exact tag identity; the binary `6e4b0e5b…1cd4` equals the independent rebuild and
reports 6.1.0, and it is now the global CLI. `pose update --no-self` changed only
the machinery manifest: `engine_version` 6.0.4→6.1.0 and the digests of the two
machinery files 6.1.0 changed, `pose-spec-closeout/SKILL.md` and
`ui-surface.md`, whose content already matched; POSE.md was skipped as the
scaffold template. A second and a third run produced no change, and
`pose check --strict` passes.

### Requirement trace

- R1 [satisfied] report:.pose/state/machinery-manifest.json — engine_version 6.1.0 recorded by the native update without force
- R2 [satisfied] check:test — repeated updates produced no change; policies, ledger and content preserved
- R3 [satisfied] check:test — `pose check --strict` passes

## 7. Final Report

### Scope delivered

This repository's own instance records the published 6.1.0 engine.

### Follow-ups

None.
