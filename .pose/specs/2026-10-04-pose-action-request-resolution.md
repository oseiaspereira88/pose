---
slug: pose-action-request-resolution
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-review-attribution-roles, pose-action-requests
priority: 0
components: pose-mcp
task_type: feature
---

# Spec: Resolve ActionRequests with attribution, authority, idempotency and subject binding

## 1. Intent

### Goal

Record answers bound to the request digest and context, validate role and assurance, and
protect resolutions against replay, conflict, retries and subject change.

### Business value

An answer received is not sufficient authorization, nor authenticated confirmation; and
persistent requests introduce concurrency that last-write-wins cannot govern.

### Constraints

Reuse the review authority contracts and their explicit limits. Corrections are
supersession/amendment events; history is never rewritten.

Program source: backlog items POSE-15 (P0, wave 2), POSE-16 (P0, wave 2) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F01, F02, F06, F08; sources
E01, E07, E08, E09, E15, E16). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

A complete IAM in POSE; guaranteeing the human's cognitive understanding; a distributed
lock service.

### Anti-mechanization guardrail

Reusar os contratos de autoridade existentes e suas limitações explícitas. Mudanças
derivadas sem relevância material não devem invalidar tudo por conveniência.

## 2. Requirements

### Functional

- R1: A resolution shall record actor, observed execution or channel, answer, timestamp, request digest, scope, attribution roles and authority evidence at the required level.
- R2: An actor without the required role shall not satisfy the request; the attempt is recorded as refused.
- R3: Declining an approval shall record the answer without authorizing the operation: answered is not satisfied.
- R4: A `confirmed_by` declared by an agent shall not pass as verified human confirmation; verified confirmation requires a signed claim bound to the request digest from an accepted issuer.
- R5: Cancelling a request shall not remove an original obligation that remains required; a waiver requires authority and a disposition the applicable contract allows.
- R6: Repeating the same operation with the same idempotency key shall not record two resolutions.
- R7: Two conflicting answers against the same expected revision shall not both be accepted; the second is refused with a conflict diagnostic and history is preserved.
- R8: A material change of question, options or subject shall make a previous answer insufficient for the new request digest.
- R9: A resolution from another project or audience shall be refused; context is revalidated before any governed effect.

### Non-functional

- File-level lock and expected-revision check reuse the spec transfer lock primitives.

### Security

- Resolution input is untrusted; signatures verified with the existing review authority verifier.

### Compatibility

- Declared mode works locally with its limitation labelled.

## 3. Technical Plan

### Affected areas

Action request journal, resolution writer, authority verification, CLI/MCP resolve
tools.

### Artifacts

- created: .pose/specs/2026-10-04-pose-action-request-resolution.md
- created: pose-mcp/internal/pose/action_resolution.go
- created: pose-mcp/internal/pose/action_resolution_test.go
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- created: .pose/changelogs/unreleased/pose-action-request-resolution.md

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Complexity of authority paths; reuse of existing verifier keeps one implementation.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-action-request-resolution`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Negative tests first: wrong role, decline, agent-declared human confirmation, cancel
without authority, replay, conflict, stale digest, foreign project. Then positive
verified path with a test issuer.

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
- Command: `pose lint-spec pose-action-request-resolution --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
