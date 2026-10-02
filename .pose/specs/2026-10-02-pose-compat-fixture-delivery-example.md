---
slug: pose-compat-fixture-delivery-example
status: done
created_at: 2026-10-02
completed_at: 2026-10-02
priority: 0
components: pose-mcp
task_type: bugfix
changelog: none
delivers: governance:compat-fixture-delivery-example
---

# Spec: Initialize the legacy compatibility fixture without a delivery example

## 1. Intent

Measure supported upgrades on a valid populated synthetic spec. The 1.1.0
new-spec template leaves an active Portuguese sample target with an empty
frontmatter delivers field; the current strict validator correctly rejects it.

Keep real upgrade behavior and strict delivery validation unchanged. Preserve
all prior-version authentication and every supported upgrade pair.

## 2. Requirements

- R1: Before upgrading the synthetic upgrade-lab-fixture, remove only the exact
  legacy Portuguese example when its delivers field is empty.
- R2: Preserve authored/nonempty delivery refs and unrelated spec bytes.
- R3: Run the full compatibility harness with all eleven prior releases; keep
  strict check, idempotent reapply and user-data preservation assertions active.

## 3. Technical Plan

Normalize the generated fixture inside compat.sh before upgrade. Match the
fixture slug and empty delivers field; never normalize real repositories.

### Artifacts

- created: .pose/specs/2026-10-02-pose-compat-fixture-delivery-example.md
- modified: tests/release/compat.sh

### Delivery targets

- governance:compat-fixture-delivery-example module:pose-mcp profile:release-governance entrypoint:tests/release/compat.sh

## 4. Tasks

- [x] Reproduce the 1.1.0 strict delivery mismatch with authenticated binaries.
- [x] Normalize only the synthetic template example before candidate upgrade.
- [x] Pass full compatibility, canonical validation and governed closeout.

## 5. Decisions

Correct the harness input instead of relaxing a validator that correctly finds
inconsistent target declarations. No public contract changes and no ADR needed.

## 6. Validation

Run `bash tests/release/compat.sh v6.2.0`, canonical strict validation and
artifact/surface gates. Verify empty-field normalization, retention of nonempty
refs and unrelated bytes, and that the same normalization clears the reproduced
legacy fixture's strict error.

### Execution log

2026-10-02: canonical strict matrix passed 48/48 at 552542f. Compatibility
harness passed all five contract/installer checks and all eleven authenticated
populated-instance upgrades. Fixture boundary checks passed five negative/positive
cases: empty field, commented empty field, nonempty refs, another slug and a
commented example. Public-claims checked 17 surfaces with zero findings.

2026-10-02: ten upgrade pairs passed before the fix. Authenticated 1.1.0 failed
strict check with 'delivers frontmatter and Delivery targets must contain the
exact same refs'. Candidate version/public metadata contracts passed after the
separate action-runtime record refresh.

### Requirement trace

- R1 [satisfied] governance:compat-fixture-delivery-example evidence:integration check:delivery-integration
- R2 [satisfied] governance:compat-fixture-delivery-example report:compatibility-report.md
- R3 [satisfied] governance:compat-fixture-delivery-example report:compatibility-report.md

## 7. Final Report

Closed on 2026-10-02 with fresh approved bundle `rvb-07d0fe5d38e43c93`,
explicit attestation and guarded lifecycle transition. Five fixture-boundary
cases and all eleven supported upgrades passed. No follow-ups introduced.
