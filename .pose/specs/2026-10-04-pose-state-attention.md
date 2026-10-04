---
slug: pose-state-attention
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-effective-governance-projection, pose-obligation-projection
priority: 1
components: pose-mcp
delivers: surface:state-attention
task_type: feature
---

# Spec: Expose Attention and obligations through one logic for CLI and MCP

## 1. Intent

### Goal

Evolve `pose state` and `pose_project_state` with an Attention view and add one
structured MCP obligations query, both fed by the same domain function.

### Business value

The user needs to find what depends on them; the agent needs causes and refs without
rebuilding them from strings.

### Constraints

No top-level command per kind of pending item; the query never mutates; ordering is
presentation, not causality.

Program source: backlog items POSE-13 (P1, wave 1) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F06, F08; sources
E06, E07, E08, E31). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Email or Slack notifications; mutation during the query.

### Anti-mechanization guardrail

Evitar uma ferramenta top-level diferente para cada espécie de pendência.

## 2. Requirements

### Functional

- R1: `pose state --attention [--scope <ref>] [--actor <id|role>]` shall show, first, items requiring that actor or role, then each item's current phase effect, what may continue, closeout and release obligations, and coverage limitations.
- R2: An MCP tool `pose_obligations` with filters scope, actor, kind, phase and state shall return the same IDs, sources and effects as the CLI for the same snapshot.
- R3: Attention shall distinguish residual debt, material actor action and mandatory gate.
- R4: An empty list with incomplete coverage shall display the limitation before any suggestion to continue.
- R5: Each item shall render actor/role, question or condition, target and restricted phase.

### Non-functional

- Golden tests for human render and JSON; catalog golden updated.

### Security

- MCP calls require `project_id` as other federated tools do.

### Compatibility

- `pose state` without the flag renders as before.

## 3. Technical Plan

### Affected areas

State CLI, MCP catalog/server, rendering, manual.

### Artifacts

- created: .pose/specs/2026-10-04-pose-state-attention.md
- created: pose-mcp/internal/pose/attention.go
- created: pose-mcp/internal/cli/state_attention.go
- created: pose-mcp/internal/cli/state_attention_test.go
- modified: pose-mcp/internal/cli/state.go
- created: pose-mcp/internal/mcpserver/obligations_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/obligations.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-state-attention.md

Reconciled against the tree at activation.

### Delivery targets

- surface:state-attention module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- UI read as gate; docs and render say Attention is not enforcement.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-state-attention`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Parity test CLI vs MCP on one fixture snapshot; incomplete-coverage render test; actor
filter test.

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
- Command: `pose lint-spec pose-state-attention --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
