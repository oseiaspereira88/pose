---
slug: pose-qualified-artifact-resolution
status: draft
created_at: 2026-09-21
completed_at:
supersedes:
depends_on: 
priority: 0
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Qualified artifact identity and one cross-project resolver

## 1. Intent

### Goal / Objetivo
Resolve the same qualified artifact through every POSE interface regardless of checkout layout or supported spec file format.

### Business value / Valor
Prevent duplicate execution and misleading closure when work crosses project boundaries.

### Constraints / Restrições
Owner: @pose-maintainers. Sole roadmap membership: [pose-multirepo-foundation](../roadmaps/pose-multirepo-foundation.md).
Extend pose-cross-repo-portfolio and project-scope contracts without reopening their historical specs. Component pose-mcp is high criticality. Consulted historical knowledge:contract-baseline-handoff; its expired narrative is context only, not current evidence.
This document plans implementation; no runtime requirement is satisfied by its creation.

### Non-goals / Não-objetivos
Do not mirror specs, auto-approve composition, introduce a hosted authority,
change unrelated product behavior or rewrite historical attestations.

## 2. Requirements

### Functional / Funcionais
- R1: Normalize local refs, legacy xref spec refs and the typed qualified forms defined in the ADR into project/kind/slug identity; reject malformed kinds, traversal and ambiguous bindings deterministically.
- R2: Use canonical Store enumeration and lookup for flat dated, flat legacy, dated folder, undated folder and split specs. Replace portfolio-specific glob/direct-path readers; listing and lookup must agree.
- R3: Bind stable explicit project IDs through existing roots/configuration. Relocation or directory rename preserves adopted identity; conflicting IDs or multiple checkouts without selected revision yield an explicit conflict. Legacy single-project defaults remain compatible.
- R4: Make lint, strict check, index, readiness, CLI and MCP use the same reference grammar and resolver. Syntactic validity and dependency satisfaction remain distinct: an unresolved required target cannot be ready.
- R5: Reuse authorized roots without implicit ancestry/sibling scanning. Unknown, unauthorized and unavailable sources remain distinct; failed external lookup must never choose a same-slug local spec.
- R6: Publish additive, versioned projection identity, source revision/digest and resolution states. Include all supported spec layouts, preserve tombstones and distinguish advisory mtime from evidence freshness.
- R7: Keep deterministic bounded resolution with visited identities, cycle/limit diagnostics and no secrets or absolute roots in MCP/projection output. Offline local resolution remains supported.
- R8: Document capability/version negotiation and preserve local refs and historic bundle semantics; add schema/consumer fixtures so an older engine refuses adopted unsupported reference metadata rather than silently ignoring it.

### Security and compatibility / Segurança e compatibilidade
Treat repository content and references as untrusted. Enforce the existing
project authorization boundary before resolving or disclosing data. Preserve
local-only refs, offline operation and historic sealed contracts. Unsupported
adopted metadata must fail explicitly; no fallback that weakens required evidence.
Bound graph traversal and input size; report stable errors without secrets/roots.

## 3. Technical Plan

### Affected areas / Áreas afetadas
Extend pose/spec.go and roots.go; centralize reference parsing and resolution in internal/pose. CLI check/lintspec/portfolio and Store readiness consume it. Update MCP schemas at the existing catalog boundary; avoid another registry.

### Artifacts
- created: .pose/specs/2026-09-21-pose-qualified-artifact-resolution.md
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/pose/roots.go
- modified: pose-mcp/internal/pose/readiness.go
- created: pose-mcp/internal/pose/artifact_ref.go
- created: pose-mcp/internal/pose/artifact_ref_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/portfolio_projection.go
- modified: pose-mcp/internal/cli/portfolio_projection_test.go
- modified: pose-mcp/internal/mcpserver/project_scope_test.go
- modified: docs-site/docs/mcp.md
- modified: .pose/indexes/validation-matrix.json

Initial exact-path inventory. Reconcile against the implementation revision before
coding and record material changes through amendments. Generated output must be
attributed to its actual producing change; do not reuse this slug for the entire
program. Commit in the owning repository with `POSE-Spec: pose-qualified-artifact-resolution`.

### Delivery targets
Proposed target: `contract:qualified-artifact-resolution`, module `pose-mcp`, existing profile `api-contract`, entrypoint `pose-mcp/cmd/pose/main.go`. Register a dedicated integration producer exercising the real CLI/MCP resolver before activation.
Keep `delivers` empty while this is a draft. Before in-progress, register the
producer, declare the typed target/entrypoint and prove a negative gate case.
Do not count documentation or a test suite matching zero tests as delivery.

### Data and rollout / Dados e rollout
Version new metadata through existing schema/capability mechanisms. Preview
adoption in populated fixtures; preserve old evidence, and reject unknown
contracts on old engines. Rollback must retain audit records and never restore
two executable authorities. No policy activation or migration occurs in planning.

### Risks / Riscos
An alias based on a checkout name can silently change identity. Preserve explicit bindings and revision context; do not pretend historical xref usage already proved consistent strict checks.

## 4. Tasks

- [ ] Confirm source revisions, ADR adoption and all local/external prerequisites.
- [ ] Reconcile artifacts and register target, evidence producer and negative fixture.
- [ ] Implement the requirements incrementally with regression/contract tests.
- [ ] Update public contracts, consumers, docs/locales and generated scaffold where affected.
- [ ] Run the scenarios below plus required module checks and record evidence per R-ID.
- [ ] Obtain explicit review and governed closeout; disposition follow-ups and refresh assessments.

## 5. Decisions

- Date: 2026-09-21. Status: Proposed.
- Adopt [qualified authority and transfer](../adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md) for this scope.
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
| Layouts, identity and malformed refs; R1–R3 | `go -C pose-mcp test ./internal/pose -run TestQualifiedArtifact -count=1` | Every supported layout resolves; rename preserves ID; conflicting bindings and traversal fail. |
| CLI/check/readiness/projection parity; R2/R4/R6 | `go -C pose-mcp test ./internal/cli -run "TestQualifiedArtifact|TestPortfolioProjection|TestXref" -count=1` | Same identity/state in every interface; dated ABM-shaped fixtures visible. |
| Authorization, cycles, old contracts; R5/R7/R8 | `go -C pose-mcp test ./internal/pose ./internal/mcpserver -run "TestQualifiedArtifact|TestProject" -count=1` | Unauthorized/missing/ambiguous projects fail without metadata leakage or fallback. |
| Required module matrix; all requirements | `pose validate --strict --module pose-mcp --report` | Registered build/unit/integration checks pass; evidence names this corpus. |

### Governance checks
- `pose lint-spec pose-qualified-artifact-resolution --ready-check` before activation.
- `pose lint-spec pose-qualified-artifact-resolution --strict` and `pose check --strict` for structure.
- `pose artifact-check --spec pose-qualified-artifact-resolution --strict` after correct Git attribution.
- `pose surface-check --spec pose-qualified-artifact-resolution --strict` after target composition.
- `pose review verify spec:pose-qualified-artifact-resolution` and `pose closeout-check spec:pose-qualified-artifact-resolution` before `pose close spec:pose-qualified-artifact-resolution`.
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
An alias based on a checkout name can silently change identity. Preserve explicit bindings and revision context; do not pretend historical xref usage already proved consistent strict checks.

### Follow-ups
No additional unowned follow-ups; remaining work is in this spec.
