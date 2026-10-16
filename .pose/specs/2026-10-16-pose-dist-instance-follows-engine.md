---
slug: pose-dist-instance-follows-engine
status: in-progress
created_at: 2026-10-16
completed_at:
depends_on: pose-review-approval-domain-preflight
priority: 0
components: docs
task_type: refactor
changelog: none
---

# Spec: Keep pose-dist's own instance on the engine it ships

## 1. Intent

### Goal
Record the engine at main (B02 merged, `268486b2`) as the engine this repository's own instance runs, and keep it there after each consolidation batch merges.

### Constraints
Run the native update without force and without self-update, with a binary built at the merged revision. The distributed POSE.md is this repository's scaffold template and is left as it is. No policy, capability or release changes.

## 2. Requirements

- R1: The native update without force records `engine_version` 7.4.1-dev, built at `268486b2bc473efee740cb513d4578755e607e9b`.
- R2: A repeated update produces no further change, and the instance's POSE.md, policies, ledgers and content are preserved.
- R3: `pose check --strict` passes, without the warning that the instance was last updated by an older engine.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-10-16-pose-dist-instance-follows-engine.md
- created: .pose/starts/pose-dist-instance-follows-engine.json
- modified: .pose/state/machinery-manifest.json

The instance was last stamped by 7.1.0 (`7eeca305`); the published 7.2.0 to 7.4.1 were not adopted here, so `pose check` warns that the instance is older than the engine. Rehearsed in a throwaway clone at `268486b2`: `pose update --no-self` changed only the manifest stamp (7.1.0 to 7.4.1-dev); the managed workflows, rules, templates and skills already matched; POSE.md was skipped (`unresolved scaffold placeholder`), which keeps the hand-maintained root manual, the source of the scaffold copies, out of the update's reach. Both the published 7.4.1 binary and the merged-engine binary read the stamped instance without warnings.

## 4. Tasks

- [x] Rehearse the update in a throwaway clone and inspect the real diff.
- [x] Apply the native update twice and verify idempotence.
- [ ] Run strict checks, obtain review and close the adoption.

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: adopt only published releases, as the 6.x adoptions did
- Selected option: stamp the instance with the engine merged at main after each consolidation batch
- Rationale: during the consolidation program the batches change the managed contract between releases; an instance stamped by 7.1.0 in the repository that ships 7.4.1-dev makes every check warn and hides a real version drift
- Consequences: the stamp carries `-dev` between releases; a release adoption still stamps the published version
- Falsifier: an update that rewrites the root POSE.md or any instance-owned content here

## 6. Validation

### Strategy
Rehearse, apply twice, then run the strict check with both binaries.

### Execution log

2026-10-16: rehearsal in a throwaway clone at `268486b2` with the binary `vcs.revision=268486b2`, `vcs.modified=false`: dry-run and apply both reported one file (`.pose/state/machinery-manifest.json`, `engine_version` 7.1.0 to 7.4.1-dev) and skipped POSE.md. `pose check --strict` went from 30 warnings, including `instance: last updated by POSE 7.1.0, older than this engine 7.4.1`, to 29 without it; the published 7.4.1 binary (`vcs.revision=7ef3be2a`) reads the stamped instance with the same 29.

### Requirement trace

- R1 [satisfied] report:.pose/state/machinery-manifest.json — engine_version 7.4.1-dev recorded by the native update without force
- R2 [satisfied] check:test — a second update produced no change; POSE.md skipped, policies, ledgers and content preserved
- R3 [satisfied] check:test — `pose check --strict` passes without the stale-instance warning

## 7. Final Report

### Delivered scope
Pending review.

### Residual risks
An update after a batch that changes managed machinery will rewrite those files; the dry-run is read before applying.

### Follow-ups
