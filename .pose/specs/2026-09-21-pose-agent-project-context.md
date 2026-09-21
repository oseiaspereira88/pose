---
slug: pose-agent-project-context
status: draft
created_at: 2026-09-21
completed_at:
supersedes:
depends_on: pose-qualified-artifact-resolution, pose-federated-roadmap-acceptance, pose-spec-authority-transfer
priority: 0
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Consistent project context for agents across repository entrypoints

## 1. Intent

### Goal / Objetivo
Make an agent starting at a coordinator or executor resolve the same qualified task and reuse its canonical spec before creation or closeout.

### Business value / Valor
Prevent duplicate execution and misleading closure when work crosses project boundaries.

### Constraints / Restrições
Owner: @pose-maintainers. Sole roadmap membership: [pose-multirepo-foundation](../roadmaps/pose-multirepo-foundation.md).
Extend existing CLI/MCP context selection rather than adding a parallel agent service. A bare ambiguous slug cannot identify an external task; require qualified intent or explicit project selection.
This document plans implementation; no runtime requirement is satisfied by its creation.

### Non-goals / Não-objetivos
Do not mirror specs, auto-approve composition, introduce a hosted authority,
change unrelated product behavior or rewrite historical attestations.

## 2. Requirements

### Functional / Funcionais
- R1: Expose selected project, qualified task, authority, revision, coordinator relation and supported contracts through existing CLI/MCP context surfaces. Avoid filesystem-root leakage in MCP.
- R2: When a task declares external authority, route authorized operations to that project regardless of cwd; when intent/project is ambiguous, stop with a diagnostic rather than creating a local shadow spec.
- R3: Before new-spec/start/close, check redirects, transfer-in-progress and declared competing authority. Reuse an existing canonical spec; unrelated projects may use the same slug without a global duplicate error.
- R4: Keep read-only context discovery separate from mutation. Cross-project writes require existing authorization for that target and compare expected context/revision at application; no implicit parent/sibling write permission.
- R5: Update feature/review/closeout skills, workflow, manual, locales and scaffold to resolve authority first, keep requirements at source, link composition and use the correct repository-scoped commit trailer.
- R6: Ensure installed binary, MCP tool context and configured project binding are validated independently. Version mismatch, changed workspace or reconnect cannot silently retain an obsolete authority.
- R7: Prove the complete two-project journey from parent root, child checkout and sibling checkout: create/reuse, execute, review, close implementation and observe coordinator still pending unmet integration.
- R8: Keep single-repository UX compatible. Publish capabilities and stable errors; add installed-distribution tests proving old engines cannot apply new transfer/federation semantics.

### Security and compatibility / Segurança e compatibilidade
Treat repository content and references as untrusted. Enforce the existing
project authorization boundary before resolving or disclosing data. Preserve
local-only refs, offline operation and historic sealed contracts. Unsupported
adopted metadata must fail explicitly; no fallback that weakens required evidence.
Bound graph traversal and input size; report stable errors without secrets/roots.

## 3. Technical Plan

### Affected areas / Áreas afetadas
Extend CLI project selection/context and MCP project-scope contract. Wire the resolver into new-spec and closeout entrypoints. Update the distributed instruction source then regenerate embedded scaffold with its existing generator.

### Artifacts
- created: .pose/specs/2026-09-21-pose-agent-project-context.md
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/cli/onboarding_context.go
- modified: pose-mcp/internal/mcpserver/mcp_context_test.go
- created: pose-mcp/internal/cli/multirepo_agent_flow_test.go
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .pose/workflows/feature.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json

Initial exact-path inventory. Reconcile against the implementation revision before
coding and record material changes through amendments. Generated output must be
attributed to its actual producing change; do not reuse this slug for the entire
program. Commit in the owning repository with `POSE-Spec: pose-agent-project-context`.

### Delivery targets
Proposed target: `surface:multirepo-agent-context`, module `pose-mcp`, existing profile `cli-surface`, entrypoint `pose-mcp/cmd/pose/main.go`. Register reachability and integration/e2e producers for the installed CLI/MCP journey.
Keep `delivers` empty while this is a draft. Before in-progress, register the
producer, declare the typed target/entrypoint and prove a negative gate case.
Do not count documentation or a test suite matching zero tests as delivery.

### Data and rollout / Dados e rollout
Version new metadata through existing schema/capability mechanisms. Preview
adoption in populated fixtures; preserve old evidence, and reject unknown
contracts on old engines. Rollback must retain audit records and never restore
two executable authorities. No policy activation or migration occurs in planning.

### Risks / Riscos
Hidden default project selection is the recurrence trigger. Explicit qualified intent must survive cwd changes; a same-slug lookup alone must not imply user intent or write authorization.

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
| Context, ambiguity, duplicate prevention; R1–R4 | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgent -count=1` | Same qualified task resolves at every entrypoint; ambiguity and partial transfers block creation. |
| Installed journey, contract mismatch; R6–R8 | `go -C pose-mcp test ./internal/cli -run TestMultiRepoAgentInstalled -count=1` | Real installed CLI/MCP route one authority and leave unmet composition blocked. |
| Instruction/scaffold parity; R5/R8 | `go -C pose-mcp test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1` | Regenerated locales/skills match canonical sources; new flow documented. |
| Required module matrix; all requirements | `pose validate --strict --module pose-mcp --report` | Registered build/unit/integration checks pass; evidence names this corpus. |

### Governance checks
- `pose lint-spec pose-agent-project-context --ready-check` before activation.
- `pose lint-spec pose-agent-project-context --strict` and `pose check --strict` for structure.
- `pose artifact-check --spec pose-agent-project-context --strict` after correct Git attribution.
- `pose surface-check --spec pose-agent-project-context --strict` after target composition.
- `pose review verify spec:pose-agent-project-context` and `pose closeout-check spec:pose-agent-project-context` before `pose close spec:pose-agent-project-context`.
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
Hidden default project selection is the recurrence trigger. Explicit qualified intent must survive cwd changes; a same-slug lookup alone must not imply user intent or write authorization.

### Follow-ups
No additional unowned follow-ups; remaining work is in this spec.
