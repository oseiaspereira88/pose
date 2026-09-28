---
slug: pose-dist-adopt-published-v6
status: in-progress
created_at: 2026-09-28
completed_at:
depends_on: pose-v6-release-readiness
priority: 0
components: docs
task_type: refactor
changelog: none
---

# Spec: Adopt the published POSE 6.0.0 machinery

## 1. Intent

Update the engine repository's operating machinery using the authenticated published 6.0.0 CLI. Preserve instance policy, contributor mode, local knowledge and the immutable release ledger. Runtime development builds remain explicitly labelled as development builds.

## 2. Requirements

- R1: The CLI used for update shall be the authenticated published 6.0.0 artifact.
- R2: Native update without force shall stamp engine_version 6.0.0 and preserve instance-owned content, policies and release evidence.
- R3: Repeating update shall leave managed files unchanged and strict structural and documentation gates shall pass.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-28-pose-dist-adopt-published-v6.md
- created: .pose/reports/2026-09-28-pose-dist-v6-adoption.md
- modified: .pose/state/machinery-manifest.json
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/pose-mcp.md
- modified: .pose/state/components/pose-mcp.json

Use native update --no-self. Inspect the actual diff before recording any additional machinery paths. The existing release tag and ledger remain immutable.

## 4. Tasks

- [x] Authenticate the published artifact and binary.
- [x] Run initial component assessment.
- [x] Deliver native machinery and verify preservation and idempotence.
- [ ] Run strict checks, obtain review and close adoption.

## 5. Decisions

Use published machinery for repository guidance and an exact revision-bound development CLI for consumer federation. A post-tag receipt commit does not change the published binary's provenance.

## 6. Validation

### Strategy
Compare native update diffs, instance-owned sections and policy/release tree hashes before and after delivery. Repeat update; run doctor, strict check, skills-check and documentation validation.

### Execution log
2026-09-28: release archive and binary hashes match the independently verified v6.0.0 ledger; Sigstore verification succeeded with the exact tag workflow identity.

### Requirement trace
- R1 [satisfied] report:.pose/reports/2026-09-28-pose-dist-v6-adoption.md archive, binary and exact Sigstore workflow identity authenticated.
- R2 [satisfied] report:.pose/reports/2026-09-28-pose-dist-v6-adoption.md native update records 6.0.0; policy, ledger, contributor mode and instance-owned sections preserved.
- R3 [satisfied] report:.pose/reports/2026-09-28-pose-dist-v6-adoption.md repeated update changes no delivered paths; strict structural, ready and skills gates pass.

## 7. Final Report

### Follow-ups
None introduced by machinery adoption. Runtime closeout defects remain canonical in dedicated engine specs.
