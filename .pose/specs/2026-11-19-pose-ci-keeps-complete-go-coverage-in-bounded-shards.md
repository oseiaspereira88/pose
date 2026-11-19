---
slug: pose-ci-keeps-complete-go-coverage-in-bounded-shards
status: in-progress
created_at: 2026-11-19
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
---

# Spec: Preserve complete engine tests in bounded Go processes

## 1. Intent

### Goal

Run the complete package test suite within bounded Go processes. The
`internal/pose` suite timed out repeatedly under Go 1.26.11, including a
hosted run with package serialization.

### Constraints

Maintain the 10-minute Go deadline for each process. Preserve the set of
packages, tests, fuzz seeds and examples executed by the default invocation.
The maintainer approved changing the aggregate scheduling budget on 2026-11-19.

## 2. Requirements

- R1: The engine runner shall execute all other packages and two disjoint halves of the internal/pose suite.
- R2: The runner shall compare Go's selected inventory with the complete inventory and refuse a duplicate or missing test.
- R3: The hosted CI, local release verifier and engine validation matrix shall invoke the same runner.

## 3. Technical Plan

### Artifacts

- created: scripts/test-engine-go.sh
- modified: .github/workflows/ci.yml
- modified: .pose/indexes/validation-matrix.json
- modified: scripts/verify.sh

## 6. Validation

### Strategy

Run `bash scripts/test-engine-go.sh inventory`, the local verifier and
matrix coverage tests, and the hosted suite. The inventory command should
report equal union and total counts. Each executed shard should use 10m.

## 7. Final Report

### Delivered scope

Validation and CI pending.
