---
slug: pose-abm-capability-adoption
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-open-backlog-reconciliation, pose-flat-spec-amendments, pose-effective-governance-projection
priority: 1
components: pose-mcp
task_type: feature
---

# Spec: Conclude the pilot and adoption decision of implemented ABM capabilities

## 1. Intent

### Goal

Reconcile the pending pilot follow-ups of contract nodes, atomic start and causality
closeout, run the planned pilot and shadow, and record an explicit adopt-or-defer
decision per capability.

### Business value

The three capabilities have implementation; the remaining work is pilot, shadow and
authorized adoption, not code.

### Constraints

No second implementation; policy changes only after the applicable stop/go and competent
decision; failures may block rollout without invalidating the delivered code.

Program source: backlog items POSE-30 (P1, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F03, F10; sources
E04, E09, E19). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Enabling flags by default; assuming the analysis authorizes adoption.

### Anti-mechanization guardrail

Respeitar a decisão anterior de implementar sem rollout.

## 2. Requirements

### Functional

- R1: No second implementation shall be created for an existing capability.
- R2: The pilot and shadow foreseen in the ABM specs shall produce evidence and a disposition of their follow-ups.
- R3: Policy shall change only after the applicable stop/go and a recorded decision by the competent authority (an ActionRequest once available).
- R4: Ceremony or compatibility failures shall be visible and may stop rollout without invalidating delivered code.
- R5: Effective governance shall show the outcome per capability after the decision.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Instances not adopting keep current behaviour.

## 3. Technical Plan

### Affected areas

ABM capability follow-ups, pilot report, review policy (only after decision).

### Artifacts

- created: .pose/specs/2026-10-04-pose-abm-capability-adoption.md
- created: .pose/reports/pose-abm-capability-adoption.md
- created: .pose/results/pose-abm-capability-adoption.json
- created: .pose/actions/act-7d587a8e4f3bf0fc.jsonl
- created: .pose/actions/act-fa1d72f029d567a0.jsonl
- created: .pose/actions/act-08a9fd2d9a0e50bb.jsonl

Reconciled against the tree at activation.

### Technical risks

- Adoption by inertia; explicit decision record.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-abm-capability-adoption`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Shadow comparison of causality closeout against current gate on recent specs; atomic
start and contract nodes on the next real specs.

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
- Command: `pose lint-spec pose-abm-capability-adoption --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

A shadow on five disposable clones at ce17db6 (report `.pose/reports/pose-abm-capability-adoption.md`) measured each capability against an unadopted baseline. Contract nodes had no observable effect in this corpus. Atomic start added 39 closeout-restricting `start-reconciliation` obligations, one per spec already in progress, with no command to record a legacy baseline and no adoption cutoff; that stops its rollout without invalidating the code. Causality closeout stamped its contract on all 39 prepared bundles; its attestation cost was not measured. No implementation was added. The three adoption decisions are open as action requests act-7d587a8e4f3bf0fc, act-fa1d72f029d567a0 and act-08a9fd2d9a0e50bb, addressed to human:oseias with the agent's recommendation (adopt, defer, defer); the review policy changes only after each answer.

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

- [open] Atomic start needs a baseline command for specs already in progress, or an adoption cutoff, before it can be adopted: the shadow measured 39 closeout-blocking reconciliations with no remedy (owner:@pose-maintainers crit:medium review:2026-11-01)
