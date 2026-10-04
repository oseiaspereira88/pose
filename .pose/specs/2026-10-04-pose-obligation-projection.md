---
slug: pose-obligation-projection
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-obligation-contract, pose-typed-producer-diagnostics
priority: 0
components: pose-mcp
task_type: feature
---

# Spec: Aggregate obligations read-only with snapshot, freshness and coverage

## 1. Intent

### Goal

Provide one deterministic, read-only aggregation of obligations from readiness, review,
closeout and start, bound to a snapshot and reporting per-producer coverage.

### Business value

One query must centralise pending items without duplicating the sources; and an empty
list from stale or unavailable sources must never mean no obligations.

### Constraints

The query changes nothing; follow-ups appear as residual attention, non-blocking by
default; no expensive silent refresh and no external call.

Program source: backlog items POSE-11 (P1, wave 1), POSE-12 (P0, wave 1) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F08, F10; sources
E06, E07, E08, E09, E15, E17, E22, E29, E31). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Persisting each pending item as a file; a `resolve_obligation` tool for derived
obligations; distributed polling.

### Anti-mechanization guardrail

Attention de follow-up não o transforma em gate de entrega por padrão. Freshness não é
um refresh automático irrestrito ou uma espera escondida.

## 2. Requirements

### Functional

- R1: The aggregation shall not modify specs, reviews, state or policy; a test hashes the tree before and after.
- R2: Resolving a condition at its source shall satisfy or remove the projected item without any other write.
- R3: An item reported by two producers shall be correlated by logical ID without erasing distinct restrictions.
- R4: Each item shall link to its source and reproduce its reason (source payload reference and code).
- R5: The answer shall carry a snapshot: project, authority context revision, source revision (commit or working-tree digest), policy digest and generation time.
- R6: Coverage shall be reported per producer as current, stale, unavailable or unsupported; a producer failure appears in coverage and never as zero items.
- R7: A changed policy or authority context shall invalidate confidence in a previous answer; working tree and index at different revisions are identified, not presented as one coherent snapshot.
- R8: Filters by scope, actor, kind, phase and state shall be supported deterministically.

### Non-functional

- Bounded cost: reuse parsed state; target under one second on the pose-dist corpus, measured and recorded.

### Security

- Cross-project refs resolve only through authorized roots.

### Compatibility

- New surface; no existing output changes.

## 3. Technical Plan

### Affected areas

New obligation aggregator and adapters, snapshot binding, coverage.

### Artifacts

- created: .pose/specs/2026-10-04-pose-obligation-projection.md
- created: pose-mcp/internal/pose/obligation_projection.go
- created: pose-mcp/internal/pose/obligation_projection_test.go
- created: pose-mcp/internal/pose/obligation_adapters.go
- created: pose-mcp/internal/pose/obligation_snapshot.go
- created: .pose/changelogs/unreleased/pose-obligation-projection.md

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Adapters drifting from producers; adapters call producer functions, never re-implement them.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-obligation-projection`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Fixture project with one obligation of each initial kind; producer-failure injection;
tree-hash invariance; policy-change invalidation test; latency measurement on pose-dist.

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
- Command: `pose lint-spec pose-obligation-projection --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
