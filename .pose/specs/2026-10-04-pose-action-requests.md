---
slug: pose-action-requests
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-obligation-contract, pose-obligation-projection
priority: 1
components: pose-mcp
delivers: surface:action-requests
task_type: feature
---

# Spec: Persist material ActionRequests with identity and a minimal lifecycle

## 1. Intent

### Goal

Let an agent open, preview and read a material request to an actor — question, options,
consequences, recipient role, targets and per-phase effect — that survives sessions.

### Business value

Choices and external operations not represented anywhere else must survive sessions with
their intent and effect; today they become follow-ups, comments or `blocked`.

### Constraints

Opening requires valid targets and effect; an unowned request stays visible as
unassigned. No credentials stored. The caller considers authorizations already given
(D16).

Program source: backlog items POSE-14 (P1, wave 2) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F06; sources
E01, E06, E07, E15). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Storing credentials; choosing on the user's behalf; turning every question into a
request.

### Anti-mechanization guardrail

Uma ActionRequest é uma necessidade governada, não um ticket para toda dúvida.

## 2. Requirements

### Functional

- R1: `pose action open` (preview by default, `--apply` to write) and the MCP equivalent shall create a request with stable ID, request digest, origin, requested_by (principal, execution), recipient role, kind (decision, approval, input, external-operation, acceptance), question, options with consequences, recommendation when present, qualified targets and per-phase effects.
- R2: The request shall be stored in an append-only journal under `.pose/actions/` and reappear after the session ends with ID and digest preserved.
- R3: `pose action show|list` and the MCP read tool shall return requests with their derived state (open, answered, cancelled, superseded, invalidated).
- R4: Opening shall refuse requests without targets or effects, with an unknown phase, or with unqualified node refs; a missing owner is accepted and rendered as unassigned.
- R5: Open requests shall feed the obligation projection as actor-action obligations with their effects.
- R6: The manual and skills shall state the materiality criterion: a request exists only if a different answer would materially change execution, scope, authority, risk acceptance, closeout or publication, and no existing authorization already covers it.

### Non-functional

- Journal records are bounded in size; schema versioned.

### Security

- Journal content is untrusted on read; refs re-validated; no secrets fields.

### Compatibility

- New directory; instances that never open requests see no change.

## 3. Technical Plan

### Affected areas

New action request domain, journal, CLI and MCP surfaces, projection adapter, manual,
skills.

### Artifacts

- created: .pose/specs/2026-10-04-pose-action-requests.md
- created: pose-mcp/internal/pose/action_request.go
- created: pose-mcp/internal/pose/action_request_test.go
- created: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/action_test.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/obligations.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-action-requests.md

Reconciled against the tree at activation.

### Delivery targets

- surface:action-requests module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Request explosion; materiality criterion, corpus case and pilot measurement.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-action-requests`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

### Assumption A1
- Claim: material requests to actors have no authoritative home among follow-ups, depends_on or status
- Status: verified
- Evidence: report:2026-10-03-pose-consolidated-analysis
- Scope: POSE engine at 392aaa5a
- Affects: R1, R2

### Decision D1
- Basis: R1, R2, R6, A1
- Minimal option: a follow-up disposition kind for pending decisions
- Selected option: a dedicated append-only ActionRequest journal projected as obligations
- Rationale: follow-ups are post-delivery debt and have no per-phase effect, recipient authority or answer binding
- Consequences: one new persisted source; materiality criterion documented to avoid request explosion
- Falsifier: the pilot shows most requests are non-material or duplicate existing authorizations


## 6. Validation

### Strategy

Round-trip across processes; refusal tests for invalid targets/effects; projection test;
corpus case for a non-material question.

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
- Command: `pose lint-spec pose-action-requests --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

The agency-readiness pilot rehearsal, run in a clone named `pilot-clone`, recorded its request as `xref:proj.pilot-clone/...`: the resolver derives the project id from the directory name without an error, and the snapshot only reported a fallback when the resolver failed. The snapshot now lists the limitation whenever no identity is declared, and `action open --apply` repeats it (stderr under `--json`). Refusing would break every project that runs the CLI without a declared identity, so the request is still recorded.

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
