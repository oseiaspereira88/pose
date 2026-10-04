---
slug: pose-governed-effect-enforcement
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-obligation-projection, pose-action-request-resolution, pose-phase-scoped-readiness
priority: 0
components: pose-mcp
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
- modified: pose-mcp/internal/pose/start.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/capabilities.go
- created: pose-mcp/internal/pose/governed_effects.go
- created: pose-mcp/internal/pose/governed_effects_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- created: .pose/changelogs/unreleased/pose-governed-effect-enforcement.md

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Over-blocking; phase-scoped effects and pilot stop/go before broad adoption.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-governed-effect-enforcement`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

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
