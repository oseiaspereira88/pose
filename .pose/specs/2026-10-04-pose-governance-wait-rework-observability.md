---
slug: pose-governance-wait-rework-observability
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-typed-producer-diagnostics, pose-action-request-resolution, pose-phase-scoped-readiness, pose-recoverable-closeout-plan
priority: 2
components: pose-mcp
delivers: surface:governance-wait-rework-observability
task_type: feature
---

# Spec: Observe pending age, waiting intervals, governance rework and computational cost

## 1. Intent

### Goal

Record minimal temporal events of requests and resolutions, report pending age, observed
waiting and known blocking separately, classify invalidation and supersession causes,
and benchmark index/read/closeout on fixed corpora.

### Business value

Telemetry has no wait field and increments `WaitDurationUnknown`; 6.3.0 had 16
attestations with half superseded, and the count alone cannot say whether that was valid
discovery, subject change or sequencing error.

### Constraints

No inferred telemetry; unknown stays unknown; no aggregate efficiency score or ranking.

Program source: backlog items POSE-26 (P2, wave 3), POSE-27 (P2, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F09, F12; sources
E01, E17, E20). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

A personal productivity tracker; a universal target of fewer reviews; promising speedup
from smaller files.

### Anti-mechanization guardrail

Não inferir telemetria que nunca foi observada. Métricas orientam investigação, sem
ranking agregado.

## 2. Requirements

### Functional

- R1: A request open for six hours shall not be counted as six hours of total inactivity; age, attributed waiting and known full blocking are separate metrics.
- R2: Simultaneous intervals shall be deduplicated in total-time aggregation.
- R3: Missing timestamps or unobserved origin events shall remain unknown.
- R4: Rework caused by subject change shall be separated from operational error or review conflict when evidence exists; unknown causes are not attributed to user or engine.
- R5: Benchmarks shall use the same inputs and report time and bytes separately, with corpus and version recorded.
- R6: `pose stats governance` shall expose these dimensions with scope, cause and coverage, without interpreting many operations as bad governance.

### Non-functional

- None specific beyond the shared constraints.

### Security

- No personal data beyond declared principals.

### Compatibility

- Existing stats fields unchanged.

## 3. Technical Plan

### Affected areas

Governance outcomes, stats CLI, benchmark scripts.

### Artifacts

- created: .pose/specs/2026-10-04-pose-governance-wait-rework-observability.md
- created: pose-mcp/internal/pose/governance_waits.go
- created: pose-mcp/internal/pose/governance_waits_test.go
- modified: pose-mcp/internal/cli/governance_stats.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/reuse_equivalence_test.go
- created: scripts/bench-governance.sh
- modified: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/cli/governance_dimensions_test.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-governance-wait-rework-observability.md

Reconciled against the tree at activation.

### Delivery targets

- surface:governance-wait-rework-observability module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Goodhart; dimensions only, documented interpretation limits.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-governance-wait-rework-observability`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Interval dedup tests; unknown preservation; cause taxonomy fixtures from the 6.3.0
superseded attestations.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./...`
- Scope: engine packages touched by this spec
- Expected: pass, including the new negative tests

#### Lint
- Command: `cd pose-mcp && go vet ./...`
- Scope: pose-mcp
- Expected: no findings

#### Security / Contract
- Command: `pose lint-spec pose-governance-wait-rework-observability --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:governance-wait-rework-observability evidence:integration test:TestWaitsSeparateAgeAttributedWaitAndKnownBlocking check:governance-waits-integration check:governance-waits-reachability
- R2 [satisfied] test:TestWaitsSeparateAgeAttributedWaitAndKnownBlocking check:governance-waits-integration
- R3 [satisfied] test:TestAMissingTimestampStaysUnknown check:governance-waits-integration
- R4 [satisfied] test:TestReworkClassifiesByWhatChangedAndKeepsUnknown check:governance-waits-integration
- R5 [satisfied] <scripts/bench-governance.sh records corpus revision and engine version and reports time and index bytes separately; run recorded in the Final Report>
- R6 [satisfied] test:TestWaitsAloneDoesNotComputeOutcomes check:governance-waits-integration

## 7. Final Report

The agency-readiness pilot rehearsal measured `stats governance --waits` at 88 s: it recomputed the whole outcomes report before printing the waits. `--waits` and `--rework` now compute only their dimensions (11 ms on the rehearsal clone); `--outcomes` adds the outcomes when wanted.

Measured on this repository at 6e5f4d9 with `scripts/bench-governance.sh 1`: `state --attention` 1.9 s, `check --strict` 83.7 s, `index` 60.0 s, delivery index 5.09 MB. The pre-batch engine (133478d) takes the same 82.8 s for `check --strict` as the current one, so the batch did not regress it; the cost predates this roadmap. On the same corpus `stats governance --rework` reported 1307 sealed bundles over 206 scopes: 477 superseded because the subject changed, 571 because the review plan changed, 17 for intent or plan, 33 for evidence, 3 unknown; of 1301 attestations, 8 extra ones followed an attestation invalid against today's policy and 36 have no recorded cause.

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
