---
slug: pose-abm-retrospective-replay
status: in-progress
created_at: 2026-09-28
completed_at:
depends_on: pose-abm-review-soundness-residuals, pose-abm-structural-delta, pose-abm-causality-attestation
priority: 1
components: pose-mcp
task_type: feature
delivers: governance:abm-retrospective-replay
---

# Spec: Read-only retrospective ABM replay

## 1. Intent

Produce the canonical source implementation for Harne8's
`xref:proj.harne8/spec:pose-abm-field-pilot` R1/R2/R9. Existing artifacts retain
their frozen contracts; a counterfactual is an observation, never a migration,
approval or rejection of historical authority. Use
knowledge:module-metadata-discovery-invalidates-review-provenance.

## 2. Requirements

- R1: Expose `pose stats replay --json` over every bounded local attestation and
  spec; retain counts of invalid, missing and truncated inputs.
- R2: Compare frozen verification with hypothetical explicit-judgment and
  structural-causality invariants without rewriting bundles or attestations.
- R3: Aggregate rejection reasons, evidence affected, structural observations,
  untraced facts and unbaselined nodes, with explicit denominators and unknowns.
- R4: Never emit principal identities, source content or filesystem roots.
  Refuse symlinks and bound artifact work. Remain offline and read-only.
- R5: Exercise at least 24 deterministic adjudicated cases spanning integrity,
  trivial proportionality, supported and unsupported structure, post-hoc and
  policy downgrade. Keep golden fixtures distinct from observed dogfood data.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-28-pose-abm-retrospective-replay.md
- created: pose-mcp/internal/pose/governance_replay.go
- created: pose-mcp/internal/pose/governance_replay_test.go
- created: pose-mcp/internal/pose/abm_golden_test.go
- created: pose-mcp/internal/pose/testdata/abm/scenarios.json
- created: pose-mcp/internal/cli/governance_replay.go
- created: pose-mcp/internal/cli/governance_replay_test.go
- modified: pose-mcp/internal/cli/insights.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-abm-retrospective-replay.md
- modified: docs-site/docs/cli.md

### Delivery targets
- governance:abm-retrospective-replay module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

Reuse the existing classifiers, confined immutable loaders and structural
detectors. Only add contracts to an in-memory copy for counterfactual checking.
Compare new blockers with frozen blockers and classify reasons into a closed
vocabulary; never export raw diagnostics. Structural coverage uses the newest
sealed subject per spec and reports absent subjects as unknown. This does not
claim missing historical start baselines were recorded before execution.

## 4. Tasks

- [x] Define the source/consumer boundary and preimplementation test plan.
- [ ] Implement bounded read-only replay and CLI.
- [ ] Implement the adjudicated golden corpus and negative contracts.
- [ ] Validate, review and close.

## 5. Decisions

Keep experimental engineering judgments opt-in if retrospective utility is
inconclusive; mechanical integrity fixes remain required. This dataset cannot
prove causal benefit or external adoption. The coordinator owns stop/go/adjust.

## 6. Validation

Before implementing, require tests proving byte-identical repository artifacts
before/after replay; missing and malformed inputs cannot count as passed; old
blank judgment rationale becomes a hypothetical rejection without changing the
historical verdict; supported/unsupported structural coverage stays separate;
limits and symlinks report incompleteness; the real CLI reaches this projector.
Register `abm-retrospective-replay-integration` and `abm-replay-reachability`,
then run the full source matrix. Golden cases have explicit expected outcomes.

## 7. Final Report

Implementation and observed corpus reports remain pending.

### Follow-ups

Prospective user observation belongs to the coordinator's owned postrelease
follow-up; no user acceptance is claimed by this source spec.
