---
slug: pose-abm-retrospective-replay
status: done
created_at: 2026-09-28
completed_at: 2026-09-28
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
- [x] Implement bounded read-only replay and CLI.
- [x] Implement the adjudicated golden corpus and negative contracts.
- [x] Validate, review and close.

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

### Requirement trace

- R1 [satisfied] reachability:pose-mcp/go/abm-replay-reachability
- R2 [satisfied] integration:pose-mcp/go/abm-retrospective-replay-integration
- R3 [satisfied] integration:pose-mcp/go/abm-retrospective-replay-integration
- R4 [satisfied] integration:pose-mcp/go/abm-retrospective-replay-integration
- R5 [satisfied] integration:pose-mcp/go/abm-retrospective-replay-integration

## 7. Final Report

Delivered the bounded read-only CLI projector and 24 adjudicated synthetic
golden cases. The full source matrix passed 36/36 at `e344a45`. Byte snapshots,
invalid input, symlinks, limits, sanitized output and actual CLI routing passed.
The separate source review approved the implementation against sealed evidence.

Before approval renewals, the observed source corpus contained 919 attestations
(918 approving decisions); four of 440 frozen record passes were additionally
rejected by the hypothetical contracts. Harne8 contained 166 approving decisions;
13 of 151 frozen record passes were additionally rejected. These record-level
observations do not apply lifecycle grandfathering or rescind historical authority.
Unknown subjects and missing telemetry remain explicit; no causal benefit,
external adoption or historical pre-start baseline is claimed.

### Follow-ups

Prospective user observation belongs to the coordinator's owned postrelease
follow-up; no user acceptance is claimed by this source spec.
