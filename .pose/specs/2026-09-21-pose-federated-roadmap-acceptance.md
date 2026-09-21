---
slug: pose-federated-roadmap-acceptance
status: draft
created_at: 2026-09-21
completed_at:
supersedes:
depends_on: pose-qualified-artifact-resolution
priority: 0
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Federated roadmap composition and verifiable acceptance

## 1. Intent

### Goal / Objetivo
Compose external specs, milestones and optional sub-roadmaps through the existing governance gates, without duplicating implementation specs.

### Business value / Valor
Prevent duplicate execution and misleading closure when work crosses project boundaries.

### Constraints / Restrições
Owner: @pose-maintainers. Sole roadmap membership: [pose-multirepo-foundation](../roadmaps/pose-multirepo-foundation.md).
Depends on qualified resolution. This scope owns engine composition only; consumer rollout has its own roadmap and must not become a bootstrap prerequisite.
This document plans implementation; no runtime requirement is satisfied by its creation.

### Non-goals / Não-objetivos
Do not mirror specs, auto-approve composition, introduce a hosted authority,
change unrelated product behavior or rewrite historical attestations.

## 2. Requirements

### Functional / Funcionais
- R1: Represent local/external spec membership, milestone prerequisites and roadmap outcome consumption with qualified refs. Preserve local syntax and require explicit adoption for new fields.
- R2: Enforce one owning active roadmap per executable spec across the authorized closure; consumption does not acquire ownership. Deduplicate progress and reject mixed cross-project cycles, including redirects and self-reference.
- R3: Seal a dependency manifest containing identities, selected immutable revisions, source contract/policy and subject/plan/content digests plus evidence/verifier results. Resolve one consistent bounded snapshot.
- R4: Require valid source review/delivery evidence and consumer trust acceptance for required external children. A done status or copied attestation alone must fail. Preserve explicit local composition criteria and judgment.
- R5: Recheck pinned inputs at close application and invalidate affected coordinator reviews on material change, authorization revocation or evidence supersession. Cache keys include project, revision, contract, policy and authorization scope.
- R6: Report unknown, unavailable, unauthorized, stale, conflicting and unsupported-contract distinctly; every unresolved required edge blocks. Offline cached proof is accepted only if adopted policy and complete trust/evidence permit verification.
- R7: Handle nested submodules and sibling repos identically; consumer adoption claims bind to its gitlink or explicitly selected dependency revision. Do not silently accept a newer child HEAD than the version consumed.
- R8: Expose the same plan, readiness and blockers in CLI/MCP/index/projection. Keep historical bundles under their sealed semantics; version schemas, preview adoption and retain rollback that refuses unreadable new contracts.

### Security and compatibility / Segurança e compatibilidade
Treat repository content and references as untrusted. Enforce the existing
project authorization boundary before resolving or disclosing data. Preserve
local-only refs, offline operation and historic sealed contracts. Unsupported
adopted metadata must fail explicitly; no fallback that weakens required evidence.
Bound graph traversal and input size; report stable errors without secrets/roots.

## 3. Technical Plan

### Affected areas / Áreas afetadas
Extend Roadmap/Milestone models, readiness, roadmap checks and review manifests. Keep one source verifier and local outcome review; use bounded recursion and existing evidence classes. Do not fetch networks or mutate child repositories during read/check.

### Artifacts
- created: .pose/specs/2026-09-21-pose-federated-roadmap-acceptance.md
- modified: pose-mcp/internal/pose/roadmaps.go
- modified: pose-mcp/internal/pose/readiness.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/portfolio_projection.go
- created: pose-mcp/internal/pose/federated_acceptance_test.go
- modified: pose-mcp/internal/cli/roadmap_check_gate_test.go
- modified: .pose/indexes/validation-matrix.json

Initial exact-path inventory. Reconcile against the implementation revision before
coding and record material changes through amendments. Generated output must be
attributed to its actual producing change; do not reuse this slug for the entire
program. Commit in the owning repository with `POSE-Spec: pose-federated-roadmap-acceptance`.

### Delivery targets
Proposed target: `governance:federated-roadmap-acceptance`, module `pose-mcp`, existing profile `backend-go`, entrypoint `pose-mcp/cmd/pose/main.go`. Add integration producer invoking real check/review/close against two Git fixture projects.
Keep `delivers` empty while this is a draft. Before in-progress, register the
producer, declare the typed target/entrypoint and prove a negative gate case.
Do not count documentation or a test suite matching zero tests as delivery.

### Data and rollout / Dados e rollout
Version new metadata through existing schema/capability mechanisms. Preview
adoption in populated fixtures; preserve old evidence, and reject unknown
contracts on old engines. Rollback must retain audit records and never restore
two executable authorities. No policy activation or migration occurs in planning.

### Risks / Riscos
Cycles or TOCTOU can turn a partial graph into false approval. Treat incomplete required closure as blocked, retain traversal limits and pin every verification input.

## 4. Tasks

- [ ] Confirm source revisions, ADR adoption and all local/external prerequisites.
- [ ] Reconcile artifacts and register target, evidence producer and negative fixture.
- [ ] Implement the requirements incrementally with regression/contract tests.
- [ ] Update public contracts, consumers, docs/locales and generated scaffold where affected.
- [ ] Run the scenarios below plus required module checks and record evidence per R-ID.
- [ ] Obtain explicit review and governed closeout; disposition follow-ups and refresh assessments.

## 5. Decisions

- Date: 2026-09-21. Status: Proposed.
- Adopt [federated acceptance](../adr/2026-09-21-federated-roadmap-acceptance-with-local-evidence-authority.md) for this scope.
- Prefer the existing Store/roots/review contracts over a second authority or resolver.
- Preserve the distinction between source implementation, consumer adoption and program acceptance.

## 6. Validation

### Strategy / Estratégia
Risk: high for authority and cross-project acceptance. The named new test families
and scripts below are implementation deliverables, not checks already executed.
All rows are mandatory. Verify that each new named family exists and actually
executes; a successful command with no matching tests is insufficient evidence.
Use independent Git fixtures, including nested submodule and sibling layouts.

| Scenario / requisitos | Command (from this repository root) | Expected evidence |
| --- | --- | --- |
| Ownership, DAG and all reference kinds; R1/R2/R8 | `go -C pose-mcp test ./internal/pose ./internal/cli -run TestFederatedRoadmap -count=1` | Mixed cycles/duplicate ownership rejected; consuming an outcome does not double-count specs. |
| Evidence and freshness; R3–R6 | `go -C pose-mcp test ./internal/pose -run TestFederatedAcceptance -count=1` | Done without valid proof fails; changed digest or revoked authorization blocks application. |
| Concurrent source changes and layouts; R3/R5/R7 | `go -C pose-mcp test -race ./internal/pose ./internal/cli -run TestFederated -count=1` | Pinned snapshot consistent; wrong gitlink revision fails; nested/sibling results agree. |
| Required module matrix; all requirements | `pose validate --strict --module pose-mcp --report` | Registered build/unit/integration checks pass; evidence names this corpus. |

### Governance checks
- `pose lint-spec pose-federated-roadmap-acceptance --ready-check` before activation.
- `pose lint-spec pose-federated-roadmap-acceptance --strict` and `pose check --strict` for structure.
- `pose artifact-check --spec pose-federated-roadmap-acceptance --strict` after correct Git attribution.
- `pose surface-check --spec pose-federated-roadmap-acceptance --strict` after target composition.
- `pose review verify spec:pose-federated-roadmap-acceptance` and `pose closeout-check spec:pose-federated-roadmap-acceptance` before `pose close spec:pose-federated-roadmap-acceptance`.
- Run `pose assess discover --update-state` at implementation closeout, not to mark this plan delivered.

### Execution log / Log de execução
2026-09-21: planning only. Editorial validation is recorded in the package audit;
implementation tests, delivery evidence and per-requirement acceptance remain pending.

### Requirement trace
- R1: Pending implementation and the mapped mandatory scenario above.
- R2: Pending implementation and the mapped mandatory scenario above.
- R3: Pending implementation and the mapped mandatory scenario above.
- R4: Pending implementation and the mapped mandatory scenario above.
- R5: Pending implementation and the mapped mandatory scenario above.
- R6: Pending implementation and the mapped mandatory scenario above.
- R7: Pending implementation and the mapped mandatory scenario above.
- R8: Pending implementation and the mapped mandatory scenario above.

## 7. Final Report

### Delivered scope / Escopo entregue
Planning artifact only. Runtime, adoption and acceptance remain pending.

### Residual risks / Riscos residuais
Cycles or TOCTOU can turn a partial graph into false approval. Treat incomplete required closure as blocked, retain traversal limits and pin every verification input.

### Follow-ups
No additional unowned follow-ups; remaining work is in this spec.
