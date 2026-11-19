---
slug: pose-check-index-regression-reads-structured-severity
status: in-progress
created_at: 2026-11-18
completed_at:
depends_on: pose-commit-gate-refuses-index-without-descriptor
components: pose-mcp
task_type: bugfix
surface: minimal
---

# Spec: Verify index gate severity through its JSON contract

## 1. Intent

### Goal
Make the index generation regression test check the JSON severity of the finding in tolerant mode.

### Business value
The pose-dist CI and SARIF jobs at f7b2fcc7 failed because the test expected the ASCII warning glyph while the runner used UTF-8. The gate itself emitted a warning for the relevant finding in both environments.

### Constraints
Preserve the human remediation assertions, both strict and tolerant modes and the index-generation contract.

## 2. Requirements

- R1: The test shall assert the machine-readable warning severity when either an index file diverges or its generation is unavailable.
- R2: The same test shall pass with C and C.UTF-8 when the expected severity is warning.

## 3. Technical Plan

### Artifacts
- modified: pose-mcp/internal/cli/check_index_generation_test.go

Check the existing JSON report for the matching finding in the two tolerant cases. Keep the strict tests on the human output because they verify operator-facing guidance.

## 6. Validation

### Strategy
Run the named regression test with LC_ALL=C and LC_ALL=C.UTF-8. The original test must reproduce under UTF-8 and the updated test must pass under both locales.


### Requirement trace
- R1 [satisfied] test:TestCheckRefusesAnIndexSetThatItsDescriptorDoesNotDescribe
- R2 [satisfied] check:check-index-generation-integration

### Execution log
The original test failed with LC_ALL=C.UTF-8 on f7b2fcc7. The check emitted the target warning with the UTF-8 glyph. The corrected test passed with LC_ALL=C and C.UTF-8.

## 7. Final Report

### Delivered scope
The regression uses the structured check response to identify a warning.
