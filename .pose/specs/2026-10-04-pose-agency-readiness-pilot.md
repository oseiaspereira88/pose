---
slug: pose-agency-readiness-pilot
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-state-attention, pose-action-request-resolution, pose-phase-scoped-readiness, pose-governed-effect-enforcement, pose-mechanization-adversarial-corpus
priority: 1
components: pose-mcp
task_type: feature
---

# Spec: Pilot the agency and readiness vertical slice before broad enforcement

## 1. Intent

### Goal

Run the vertical slice on real work in POSE and Harne8 — trivial change, public contract
change, human decision, external operation, refusal, authority transfer, recovery —
compare against a baseline and record a stop/go.

### Business value

Schemas and tests do not show usefulness or absence of friction in a real session.

### Constraints

Adoption is explicit and independent of implementation completion. Dogfooding is
complemented by use outside the author's context when possible.

Program source: backlog items POSE-29 (P1, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F06, F08, F09, F11, F12; sources
E01, E19, E24). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Universal rollout by release number; confusing an automated demo with human experience.

### Anti-mechanization guardrail

Dogfooding deve ser complementado por uso fora do contexto do autor quando possível.

## 2. Requirements

### Functional

- R1: A real case shall go through open request, Attention, resolve, revalidate and continue without manual search across subsystems.
- R2: A trivial change shall keep the minimal flow and a previous authorization shall not be asked again.
- R3: The pilot shall record unknowns and modelling problems without silently adjusting a gate to pass.
- R4: The baseline comparison shall report commands needed, human interventions, invalidations by cause, unknown data, time per operation and delivered result.
- R5: A stop/go record shall state rollback conditions and the delimited adoption scope; policy changes only after it.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- None specific beyond the shared constraints.

## 3. Technical Plan

### Affected areas

Pilot report and results; no engine code unless defects are found (each defect gets its
own spec or amendment).

### Artifacts

- created: .pose/specs/2026-10-04-pose-agency-readiness-pilot.md
- created: .pose/reports/pose-agency-readiness-pilot.md
- created: .pose/results/pose-agency-readiness-pilot.json

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Author bias; record limitation explicitly.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-agency-readiness-pilot`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Pre-registered scenarios and baseline taken before enabling the capability.

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
- Command: `pose lint-spec pose-agency-readiness-pilot --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
