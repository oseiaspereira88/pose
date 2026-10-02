---
slug: pose-ci-release-completeness
status: in-progress
created_at: 2026-10-02
completed_at:
priority: 2
components: pose-mcp
task_type: feature
delivers: governance:ci-release-completeness
---
# Spec: Enforce the release obligations which otherwise drift by hand

## 1. Intent
### Goal
Make missing local gates, clean-tree assertions and published assets deterministic failures.
### Business value
Three manual lists can silently omit new work; a root security config also blocks legacy review because its bytes are not classified.
### Constraints
Keep existing security checks and signatures. Preserve exact subject bytes. No native package dispatch.
### Non-goals
Release publication during implementation, broad review-source classification, or replacing the workflow parser globally.

## 2. Requirements
### Functional
- R1: Derive CI shell gate commands from the current workflow and require each in the local verification script; added commands must fail the parity test.
- R2: Require a clean-tree assertion after every release run step before GoReleaser, including version resolution; inserted unpaired steps must fail.
- R3: Independently require all six archives, their SBOM/signature companions, checksums, configured extra files and uploaded package manifests in the provider asset set; missing assets must fail even if no documentation references them.
- R4: Seal exact root .gitleaks.toml bytes as governed security configuration; keep secret env files, nested similarly named files and unknown TOML unclassified.
### Compatibility
No CLI schema or release tag changes. The verified asset set is the existing published contract.
### Security
No bypass for credentials, no broad extension-based classification.

## 3. Technical Plan
### Artifacts
- created: tests/release/published-asset-set.py
- created: tests/release/release-obligations.py
- modified: .github/workflows/release.yml
- modified: .github/workflows/verify-release.yml
- modified: pose-mcp/internal/version/release_harness_discovery_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- created: .pose/changelogs/unreleased/pose-ci-release-completeness.md
### Delivery targets
- governance:ci-release-completeness module:pose-mcp profile:release-governance entrypoint:.github/workflows/verify-release.yml
### API/contract changes
No public structural change.
### Technical risks
Tests fail closed on unsupported workflow forms; retain production and independent verification as separate facts.

## 4. Tasks
### Implementation
- [x] Implement derived checks with negative controls.
### Validation
- [x] Run obligation, asset and byte-retention regressions.

## 5. Decisions
### Decision 1
- Date: 2026-10-02
- Decision: enforce the repository's current explicit step syntax and reject unsupported forms; defer a whole-engine YAML parser change.
- Rationale: the new checks cover bounded provider scripts, not all project workflow dialects.

## 6. Validation
### Strategy
Apply pose-test-plan: mutate copies of the CI/release workflows to add an omitted gate or unpaired step; remove each required release asset; prove exact security-config bytes are sealed while negative paths remain blocked.
### Deterministic checks
| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Gate parity, clean-tree insertion, missing asset (required) | python3 tests/release/release-obligations.py | Positive fixture passes; all three omission cases fail |
| Real published inventory (required) | python3 tests/release/published-asset-set.py v6.2.0 | Current provider asset set complete |
| Classification boundary (required) | go -C pose-mcp test ./internal/pose -run ReviewBundle -count=1 | Byte retention and negative path tests pass |
| Canonical matrix (required) | pose validate --strict --module pose-mcp --json-out .pose/results/delivery-validation.json --report | All checks pass |
### Execution log
Canonical module validation passed 50/50 on 2026-10-02. Obligation mutations and all 35 single-asset removals were rejected. The independently retrieved v6.2.0 provider inventory satisfied the full asset contract; byte-retention regressions and ShellCheck passed.
### Requirement trace
- R1: check:ci-release-obligations-integration — TestLocalVerifyCoversCurrentCIGates rejects newly omitted shell, POSE and Go gates.
- R2: check:ci-release-obligations-integration — an inserted release step without a following clean-tree assertion fails.
- R3: check:ci-release-obligations-integration — each required archive, signature, SBOM, checksum, extra file and package manifest is individually removed and rejected; the real v6.2.0 inventory passed.
- R4: test:TestReviewBundleSealsRootGitleaksConfigBytes and test:TestReviewBundleDoesNotGeneralizeGitleaksClassification — exact root bytes retained; secret and similarly named paths stay unclassified.
### Known gaps
No native OS execution is implied by asset presence.

## 7. Final Report
### Delivered scope
Local verification parity, clean-tree pairing and independent published asset completeness are enforced with negative controls. Exact root security configuration can be reviewed without widening secret-file classification.
### Follow-ups
None.
