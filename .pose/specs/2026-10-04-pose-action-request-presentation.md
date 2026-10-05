---
slug: pose-action-request-presentation
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-state-attention, pose-action-requests, pose-phase-scoped-readiness
priority: 2
components: pose-mcp
delivers: capability:action-request-presentation
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
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-action-request-presentation.md

Reconciled against the tree at activation.

### Delivery targets

- capability:action-request-presentation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Grouping read as joint answer; renders show per-request answer controls.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
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

### Requirement trace

- R1 [satisfied] test:TestRelatedRequestsAreGroupedWithoutMergingAnswers check:action-presentation-integration
- R2 [satisfied] <release-only requests are grouped as able to wait, with earliest_phase> test:TestRelatedRequestsAreGroupedWithoutMergingAnswers check:action-presentation-integration
- R3 [satisfied] <each item carries the request digest and revision it is answered against> test:TestRelatedRequestsAreGroupedWithoutMergingAnswers check:action-presentation-integration
- R4 [satisfied] <a satisfied request is not presented again> test:TestRelatedRequestsAreGroupedWithoutMergingAnswers check:action-presentation-integration
- R5 [satisfied] <pose_action_requests with present:true> test:TestToolsList check:action-presentation-integration test:TestCatalogGovernanceBijection

## 7. Final Report

### Delivered scope

Delivers grouping for one conversation on CLI and MCP without merging answers. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
