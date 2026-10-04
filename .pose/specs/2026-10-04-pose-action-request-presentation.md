---
slug: pose-action-request-presentation
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-state-attention, pose-action-requests, pose-phase-scoped-readiness
priority: 2
components: pose-mcp
task_type: feature
---

# Spec: Group related requests and present concrete questions

## 1. Intent

### Goal

Present related requests together by theme and impact, with alternatives, consequences
and recommendation, as a contract Harne8 can consume to decide when to present.

### Business value

A good human loop maximizes authorized work and reduces interruptions without merging
distinct decisions.

### Constraints

Grouping is presentation only: IDs, authority and answers stay independent.

Program source: backlog items POSE-20 (P2, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F06, F11; sources
E01, E24). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Sending external messages from the core; a universal interruption policy.

### Anti-mechanization guardrail

Agrupar UI não significa auto-resolver nem ampliar autorização.

## 2. Requirements

### Functional

- R1: Three related requests shall be presentable together while keeping independent IDs and answers.
- R2: Requests restricting only release shall not demand immediate interruption of implementation by default; the presentation contract carries the earliest restricted phase.
- R3: The person shall see exactly the content the resolution will confirm (rendered from the request digest payload).
- R4: Reopening a session shall not repeat requests already satisfied for the same subject.
- R5: A machine-readable presentation bundle shall be exposed via MCP for platform consumers.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- None specific beyond the shared constraints.

## 3. Technical Plan

### Affected areas

Attention rendering, MCP presentation surface.

### Artifacts

- created: .pose/specs/2026-10-04-pose-action-request-presentation.md
- created: pose-mcp/internal/pose/action_presentation.go
- created: pose-mcp/internal/pose/action_presentation_test.go
- modified: pose-mcp/internal/cli/state_attention.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- created: .pose/changelogs/unreleased/pose-action-request-presentation.md

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Grouping read as joint answer; renders show per-request answer controls.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-action-request-presentation`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Grouping fixture; digest-equals-render test; session reopen test.

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
- Command: `pose lint-spec pose-action-request-presentation --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
