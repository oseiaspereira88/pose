---
slug: pose-obligation-contract
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-blocked-semantics-alignment
priority: 1
components: pose-mcp
task_type: feature
---

# Spec: Define the common obligation contract and qualified node references

## 1. Intent

### Goal

Accept the transversal ADR and publish a versioned read-model schema for obligations
with qualified node refs, stable logical IDs and source revisions.

### Business value

Unification must separate duty, satisfaction, knowledge, effect and authority without
creating redundant state; doing it per subsystem would reproduce the fragmentation.

### Constraints

Every mutable field names its authoritative source; the projection has no resolution
journal. `R4`/`D2` are qualified by project and artifact.

Program source: backlog items POSE-09 (P1, wave 1) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F06, F08; sources
E06, E07, E08, E09, E15). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

A mandatory requirement DAG; a scheduler or queues.

### Anti-mechanization guardrail

Derivar obrigações; persistir fatos novos somente onde forem necessários.

## 2. Requirements

### Functional

- R1: An obligation record shall carry identity (stable ID, project, source ref), duty (category, reason code, satisfaction condition, rule), recipient (actor, role or unassigned), scope (qualified refs), effect (phases and scopes restricted), observation (source revision, freshness, coverage, limitations) and satisfaction (pending, satisfied, waived, cancelled, invalidated) as the producing domain defines it.
- R2: The ID shall depend on the logical identity of the source (project, artifact, node, rule), not on message text, list order or query time; repeated queries and wording-only changes keep it.
- R3: Node refs shall be `xref:<project>/<kind>:<slug>` plus node kind and local id; `R4` of two specs or projects never collide.
- R4: `unknown` knowledge shall be representable independently of satisfaction and shall never be rendered as satisfied, cancelled or no-blocker.
- R5: The ADR `obligations-are-projected-action-requests-are-persisted` shall move to Accepted with examples for dependency, judgment, evidence, reconciliation and actor-action obligations.
- R6: A JSON schema shall be published under `pose-mcp/schemas/` and validated by tests against fixtures from each category.

### Non-functional

- Schema versioned from v1; unknown future fields are rejected by v1 readers with a diagnosable error.

### Security

- Refs are parsed with `ParseArtifactRef`; traversal and unauthorized projects are refused before lookup.

### Compatibility

- No existing output changes in this spec.

## 3. Technical Plan

### Affected areas

New obligation types, artifact ref grammar extension for node refs, schema, ADR.

### Artifacts

- created: .pose/specs/2026-10-04-pose-obligation-contract.md
- created: pose-mcp/internal/pose/obligation.go
- created: pose-mcp/internal/pose/obligation_test.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- created: pose-mcp/schemas/obligation.schema.json
- modified: .pose/adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md
- created: docs/architecture/pose-obligations-and-agency.md
- created: .pose/changelogs/unreleased/pose-obligation-contract.md

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Over-generic contract hiding local detail; each record links to its source payload.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-obligation-contract`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

### Assumption A1
- Claim: every pending-state shape in the engine can be mapped to the record without losing source-specific data
- Status: unverified
- Evidence: report:2026-10-03-pose-consolidated-analysis
- Scope: readiness, review, closeout, start, state, docs review and transfer at 392aaa5a
- Affects: R1

### Decision D1
- Basis: R1, R2, A1
- Minimal option: document the existing shapes and leave aggregation to callers
- Selected option: one versioned read model with adapters that keep a link to source-specific detail
- Rationale: callers rebuilding causal state from prose is the failure being fixed; adapters keep detail reachable
- Consequences: one schema to version; adapters added incrementally with explicit coverage
- Falsifier: the pilot shows agents still need the per-subsystem calls for most decisions


## 6. Validation

### Strategy

Schema tests per category; ID stability test across wording change and reordering;
collision test for same R-ID across specs and projects.

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
- Command: `pose lint-spec pose-obligation-contract --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
