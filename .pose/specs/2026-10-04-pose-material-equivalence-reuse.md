---
slug: pose-material-equivalence-reuse
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-obligation-projection, pose-recoverable-closeout-plan
priority: 1
components: pose-mcp
delivers: capability:material-equivalence-reuse
task_type: feature
---

# Spec: Reuse evidence and criteria by material equivalence and provenance

## 1. Intent

### Goal

Explain and govern reuse: evidence and criteria are reused only when their material
inputs are equivalent and policy allows it, citing origin and reason.

### Business value

Redoing valid observations costs time; copying a conclusion to another subject costs
integrity.

### Constraints

Neither universal prohibition of judgment reuse nor reuse by timestamp proximity.
Integrates with the existing criterion reuse.

Program source: backlog items POSE-22 (P1, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F09, F11, F12; sources
E07, E08, E17). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Reusing human approval for a different subject; caches without project binding.

### Anti-mechanization guardrail

Nem proibição universal de judgment reuse, nem reuse por proximidade de timestamp.

## 2. Requirements

### Functional

- R1: A change of relevant code, tool version or environment shall invalidate the corresponding evidence, with the changed input named.
- R2: Reformatting a derived report without material effect shall not require new judgment automatically.
- R3: Reused judgment shall require equivalence of the criterion's inputs and policy permission.
- R4: Each reuse shall cite its origin record and why it still holds (input bindings compared).
- R5: Closeout plan and review verify shall render reused/invalidated with input bindings.

### Non-functional

- None specific beyond the shared constraints.

### Security

- Reuse keys include project id.

### Compatibility

- Existing criterion reuse records remain valid.

## 3. Technical Plan

### Affected areas

Criterion reuse, evidence binding, closeout plan.

### Artifacts

- created: .pose/specs/2026-10-04-pose-material-equivalence-reuse.md
- created: pose-mcp/internal/pose/reuse_equivalence.go
- created: pose-mcp/internal/pose/reuse_equivalence_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-material-equivalence-reuse.md

Reconciled against the tree at activation.

### Delivery targets

- capability:material-equivalence-reuse module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Equivalence too loose; inputs enumerated per evidence class.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-material-equivalence-reuse`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Mutation of each input class invalidates; formatting-only change reuses; cross-project
reuse refused.

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
- Command: `pose lint-spec pose-material-equivalence-reuse --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestBookkeepingChangesKeepCriteriaEquivalentAndCodeChangesNameTheInput check:material-reuse-integration
- R2 [satisfied] test:TestBookkeepingChangesKeepCriteriaEquivalentAndCodeChangesNameTheInput check:material-reuse-integration
- R3 [satisfied] test:TestEquivalenceAgreesWithTheReuseDigest check:material-reuse-integration
- R4 [satisfied] test:TestEquivalenceAgreesWithTheReuseDigest check:material-reuse-integration
- R5 [satisfied] <verify carries delta.criterion_reuse> test:TestBookkeepingChangesKeepCriteriaEquivalentAndCodeChangesNameTheInput check:material-reuse-integration

## 7. Final Report

### Delivered scope

Delivers per-criterion input contracts and the reuse explanation on verify. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
