---
slug: pose-review-assurance-disclosure
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: 
priority: 0
components: pose-mcp
delivers: surface:review-assurance-disclosure
task_type: feature
---

# Spec: Disclose the identity assurance and separation actually verified by review

## 1. Intent

### Goal

Make every review, verify and closeout output state whether separation was required,
declared or authenticated, and never promise cognitive independence.

### Business value

Under `identity_assurance: declared` (the default when the policy is silent) separation
checks compare textual identities and prefixes; under `verified` they check signed
claims, principals, executions, issuer, audience and grants. Presenting `same-actor-
separate-execution` or `different-actor` without the mode lets a declared label read as
an authenticated fact — the reading the 6.3.0 attestations invite.

### Constraints

Explanations only; no policy is raised to `verified` to improve a label. Existing
verdicts do not change.

Program source: backlog items POSE-03 (P0, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F01, F02; sources
E01, E04, E07, E08, E16). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Proving independent thinking; requiring human review universally; changing the review
authority contract.

### Anti-mechanization guardrail

Não elevar a política de toda instância para verified apenas para melhorar o rótulo.

## 2. Requirements

### Functional

- R1: A policy without `identity_assurance` shall expose `declared` explicitly in review plan, verify, review-check and closeout outputs (human and JSON).
- R2: In declared mode, `agent:`/`human:` prefixes shall be rendered as declared identities, never as proof of a person or of a separate execution.
- R3: In verified mode, outputs shall name the claim, the binding (bundle digest, executions) and the authority grant that were verified.
- R4: Each output shall carry three separation fields: required (from the plan), declared (from the record) and verified (from the assurance path), with `not-verified` where applicable.
- R5: No output, help text or manual section shall use "independent review" for a fact the engine cannot observe; a test scans rendered catalogs for that wording.

### Non-functional

- Same domain function feeds CLI and MCP.

### Security

- A declared identity cannot be upgraded to verified by any field the attestation author controls.

### Compatibility

- New JSON fields are additive; existing fields keep their values.

## 3. Technical Plan

### Affected areas

Review bundle and attestation rendering, review verify, review-check, closeout-check,
MCP review tools, manual.

### Artifacts

- created: .pose/specs/2026-10-04-pose-review-assurance-disclosure.md
- created: pose-mcp/internal/pose/review_assurance.go
- created: pose-mcp/internal/pose/review_assurance_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/review_assurance_cli_test.go
- created: pose-mcp/internal/cli/review_independence_wording_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-review-assurance-disclosure.md

Reconciled against the tree at activation.

### Delivery targets

- surface:review-assurance-disclosure module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Label churn in golden files; bounded to additive fields and reviewed wording.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-review-assurance-disclosure`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Golden tests for declared and verified fixtures; a negative test that a declared
`human:` reviewer renders as declared; the wording scan for "independent".

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
- Command: `pose lint-spec pose-review-assurance-disclosure --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:review-assurance-disclosure evidence:integration test:TestReviewSurfacesDiscloseDeclaredAssuranceOfAHumanLabel check:review-assurance-disclosure-integration test:TestReviewCheckCarriesTheSameAssuranceAsVerify
- R2 [satisfied] test:TestDeclaredHumanReviewerIsNeverRenderedAsAVerifiedPerson check:review-assurance-disclosure-integration test:TestDeclaredIndependentAgentPrefixIsADeclaration
- R3 [satisfied] test:TestVerifiedAssuranceDisclosesTheClaimItVerified check:review-assurance-disclosure-integration test:TestVerifiedAssuranceWithoutAValidClaimVerifiesNothing
- R4 [satisfied] test:TestDeclaredHumanReviewerIsNeverRenderedAsAVerifiedPerson check:review-assurance-disclosure-integration test:TestLegacyAttemptUnderVerifiedPolicyVerifiesNothing
- R5 [satisfied] test:TestNoSurfaceClaimsAnIndependentReview check:review-assurance-disclosure-integration

## 7. Final Report

### Delivered scope

Delivers declared/verified disclosure and three separation fields on verify, check, closeout and plan outputs. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Coverage of earlier follow-ups

This spec covers the follow-up in pose-abm-review-soundness that reviewer authority is satisfied by a declared prefix while `different-actor` and `mandatory-human` assert an identity they do not verify: every output now states the identity as declared and the separation as not verified, so nothing asserts what was not checked. Requiring verified identity by default is a separate decision, taken with the Harne8 confirmation channel that would issue the claims. Confirmed by the maintainer on 2026-10-05 while resolving the backlog reconciliation (spec pose-open-backlog-reconciliation, Decision 5).

### Follow-ups

None recorded at planning time.
