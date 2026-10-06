---
slug: pose-action-request-resolution
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-review-attribution-roles, pose-action-requests
priority: 0
components: pose-mcp
delivers: surface:action-request-resolution
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
- created: pose-mcp/internal/cli/action_resolve.go
- created: pose-mcp/internal/cli/action_test.go
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/adversarial_corpus_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-action-request-resolution.md -> .pose/changelogs/v7.0.0/pose-action-request-resolution.md

Reconciled against the tree at activation.

### Delivery targets

- surface:action-request-resolution module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Complexity of authority paths; reuse of existing verifier keeps one implementation.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-action-request-resolution`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

### Decision D1
- Date: 2026-10-04
- Context: MCP exposes only read, gate and external-event tools; review sealing and attestation import are CLI writes.
- Options considered: add repository-write MCP tools for open and resolve; keep writes on the CLI and read over MCP.
- Decision: keep writes on the CLI; `pose_action_requests` reads.
- Rationale: a repository-write class would change the MCP catalog's security contract, which needs its own ADR; the trusted channel (Harne8) can drive the CLI or a future authenticated write path.
- Consequences: agents resolve through `pose action`; the MCP read returns the same views.

### Decision D2
- Date: 2026-10-04
- Context: a role must be held by an actor for an answer to count.
- Options considered: trust a role named on the command line; declare role membership in policy.
- Decision: `.pose/policy/actions.json` maps roles to principals; signing pins, the human-authority grant and the audience stay in the review policy.
- Rationale: one trust configuration; a role named by the resolver alone would let any actor claim it.
- Consequences: a role-addressed request has nobody to resolve it until the instance declares the role.

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

### Requirement trace

- R1 [satisfied] surface:action-request-resolution evidence:integration test:TestActionResolveRefusesTheWrongActorAndRecordsTheRightOne check:action-resolution-integration test:TestAVerifiedClaimBoundToTheRequestIsAccepted
- R2 [satisfied] test:TestAnActorWithoutTheRoleDoesNotSatisfyTheRequest check:action-resolution-integration
- R3 [satisfied] test:TestDecliningAnApprovalIsAnAnswerThatAuthorizesNothing check:action-resolution-integration
- R4 [satisfied] test:TestADeclaredHumanAnswerIsLabelledAndRefusedWhereVerifiedIsRequired check:action-resolution-integration test:TestAVerifiedClaimBoundToTheRequestIsAccepted
- R5 [satisfied] test:TestCancelAndWaiveNeedTheRightAuthority check:action-resolution-integration
- R6 [satisfied] test:TestReplayIsIdempotentAndConflictsAreRefused check:action-resolution-integration
- R7 [satisfied] test:TestReplayIsIdempotentAndConflictsAreRefused check:action-resolution-integration
- R8 [satisfied] test:TestAStaleDigestOrAChangedSubjectDoesNotSatisfy check:action-resolution-integration
- R9 [satisfied] test:TestAResolutionFromAnotherProjectIsRefused check:action-resolution-integration

## 7. Final Report

### Delivered scope

Delivers resolve, cancel, waive and invalidate with roles, digests, revisions, idempotency and verified claims as specified. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
