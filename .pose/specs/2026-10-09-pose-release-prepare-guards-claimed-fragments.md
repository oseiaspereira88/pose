---
slug: pose-release-prepare-guards-claimed-fragments
status: done
created_at: 2026-10-09
completed_at: 2026-10-10
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: governance:release-prepare-guards-claimed-fragments
---

# Spec: Release prepare does not break specs that claim unreleased fragments

## 1. Intent

### Goal

Stop `pose release prepare --apply` from archiving a changelog fragment that a spec still claims at its `unreleased/` path, without warning.

### Business value

The v7.1.0 freeze (`2cf8d4b1`) turned CI red on main: `pose-open-backlog-reconciliation` claimed three fragments under `.pose/changelogs/unreleased/` that it had written for other specs, and the freeze moved them to `.pose/changelogs/v7.1.0/`. `release check --strict` and `check --strict` passed locally before the commit; the error appeared only in CI's structural gate.

## 2. Requirements

### Functional

- R1: `pose release plan` and `pose release prepare` shall list every spec whose Artifacts claim a fragment path the freeze will move, other than the fragment's own spec.
- R2: `pose release prepare --apply` shall refuse while such a claim exists, naming the spec and the path, unless `--allow-moved-claims` is given.
- R3: `pose release check --strict` on the prepared tree shall fail on the same condition the CI structural gate failed on.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-release-prepare-guards-claimed-fragments.md
- created: .pose/starts/pose-release-prepare-guards-claimed-fragments.json
- created: .pose/changelogs/unreleased/pose-release-prepare-guards-claimed-fragments.md
- created: pose-mcp/internal/pose/release_claimed_fragments.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: pose-mcp/internal/cli/help_catalog.go

### Delivery targets

- governance:release-prepare-guards-claimed-fragments module:pose-mcp profile:release-governance entrypoint:scripts/release.sh

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: the guard needs each spec's Artifacts claims, which `ListSpecs` does not carry; the first version read metadata only and found nothing.
- Decision: load each spec's body (`GetSpec`) before parsing its claims.
- Rationale: measured on the tree just before the v7.1.0 freeze (2cf8d4b1^): the metadata-only version reported 0 foreign claims; the fixed one reports exactly the three that failed CI.

## 6. Validation

### Strategy

Fixture with one spec claiming another spec's unreleased fragment: plan lists it, prepare refuses, and the test fails on the current engine.

### Execution log

2026-10-10: on a worktree at 2cf8d4b1^ (pose-dist just before the v7.1.0 freeze), `ForeignFragmentClaims` over the nine unreleased fragments reports exactly three claims: pose-open-backlog-reconciliation claiming the fragments of pose-attention-projects-every-source, pose-range-names-its-other-work and pose-trace-test-refs-resolve — the three CI rejected on main after the freeze.

2026-10-10, independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): low severity. With `--allow-moved-claims` prepare froze without naming the claims, and the help did not show the flag. Prepare now prints `release.allowed.moved-claim=` for each, the test asserts it, and the help lists the flag.

### Requirement trace

- R1 [satisfied] governance:release-prepare-guards-claimed-fragments evidence:integration test:TestReleasePrepareRefusesFragmentsOtherSpecsClaim
- R2 [satisfied] governance:release-prepare-guards-claimed-fragments evidence:integration test:TestReleasePrepareRefusesFragmentsOtherSpecsClaim
- R3 [satisfied] governance:release-prepare-guards-claimed-fragments evidence:integration test:TestReleasePrepareRefusesFragmentsOtherSpecsClaim

## 7. Final Report

### Delivered scope

`release plan` names each spec that claims another spec's fragment at its unreleased path, `release prepare --apply` refuses unless `--allow-moved-claims`, and `release check` reports it on a prepared tree.

### Residual risks

None yet.

### Follow-ups
