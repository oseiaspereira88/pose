---
slug: pose-governed-effect-enforcement
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-obligation-projection, pose-action-request-resolution, pose-phase-scoped-readiness
priority: 0
components: pose-mcp
delivers: surface:governed-effects
task_type: feature
---

# Spec: Enforce governed effects at the domain write points of start, close and release

## 1. Intent

### Goal

When the capability is adopted, make start, close and release refuse transitions
restricted by unsatisfied obligations, at the Store/domain layer shared by CLI and MCP.

### Business value

A pending item shown in a UI must restrict the corresponding transition even when the UI
is bypassed.

### Constraints

Opt-in capability `agency_readiness_version`; without adoption, documented legacy
behaviour; the projection does not become a parallel policy engine — gates consult
obligations for their own phase only.

Program source: backlog items POSE-18 (P0, wave 2) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F06, F08; sources
E08, E09, E15). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Blocking every phase for every pending item; adopting the contract automatically.

### Anti-mechanization guardrail

Obligation projection não pode virar policy engine paralelo.

## 2. Requirements

### Functional

- R1: A direct domain call shall not close or publish while an adopted, unsatisfied obligation restricts that phase.
- R2: CLI and MCP shall refuse the same transition with the same structured cause.
- R3: The write path shall revalidate obligations at write time; an earlier query never authorizes a later write after source or authority changed.
- R4: An instance without adoption shall keep the legacy behaviour and receive no new gate through an engine update; effective governance shows the capability as supported and not adopted.
- R5: Preview modes shall report the would-be refusal with reason codes and write nothing.

### Non-functional

- Enforcement covered by the adversarial corpus.

### Security

- Bypass attempts (direct file edit of status) are detected as reconciliation obligations, consistent with atomic start.

### Compatibility

- Opt-in only.

## 3. Technical Plan

### Affected areas

Start, close, release lifecycle domain functions; capability policy; CLI/MCP.

### Artifacts

- created: .pose/specs/2026-10-04-pose-governed-effect-enforcement.md
- created: pose-mcp/internal/pose/governed_effects.go
- created: pose-mcp/internal/pose/governed_effects_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/diagnostic_codes.go
- modified: pose-mcp/internal/pose/diagnostic_codes_test.go
- modified: pose-mcp/internal/pose/effective_governance.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- created: pose-mcp/internal/cli/governed_effects_test.go
- modified: pose-mcp/internal/mcpserver/obligations_test.go
- modified: docs-site/docs/obligations.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-governed-effect-enforcement.md

Reconciled against the tree at activation.

### Delivery targets

- surface:governed-effects module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Over-blocking; phase-scoped effects and pilot stop/go before broad adoption.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-governed-effect-enforcement`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

### Decision D1
- Date: 2026-10-04
- Context: the closeout transition is written by the CLI `close` (and continuous closeout); MCP has no close tool, and `pose release record` records facts that already happened outside.
- Options considered: refuse at every command that reads closeout state; refuse where each transition is written; also refuse recording a publication.
- Decision: the closeout state carries the typed diagnostic (so continuous closeout and every reader see it), `close` refuses on it explicitly, start refuses through its recomputed plan, and release refuses at `prepare`, the freeze. Recording a publication that happened is not refused.
- Rationale: refusing to record an external fact would hide reality without preventing it; the freeze is the last point POSE controls.
- Consequences: execution has no write point and is never refused; it is shown in phases and Attention.

## 6. Validation

### Strategy

Parity tests domain/CLI/MCP; stale-query test; non-adopted instance golden unchanged.

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
- Command: `pose lint-spec pose-governed-effect-enforcement --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
