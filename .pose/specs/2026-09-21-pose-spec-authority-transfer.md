---
slug: pose-spec-authority-transfer
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

# Spec: Explicit and recoverable spec authority transfer

## 1. Intent

### Goal / Objetivo
Move authority or reconcile an existing executor without creating two executable specs or rewriting historical delivery evidence.

### Business value / Valor
Prevent duplicate execution and misleading closure when work crosses project boundaries.

### Constraints / Restrições
Owner: @pose-maintainers. Sole roadmap membership: [pose-multirepo-foundation](../roadmaps/pose-multirepo-foundation.md).
Transfer is an explicit authorized mutation, never a side effect of lookup. Decomposition is a separate operation with distinct scoped specs. No distributed Git transaction or automatic commits/pushes.
This document plans implementation; no runtime requirement is satisfied by its creation.

### Non-goals / Não-objetivos
Do not mirror specs, auto-approve composition, introduce a hosted authority,
change unrelated product behavior or rewrite historical attestations.

## 2. Requirements

### Functional / Funcionais
- R1: Produce a read-only deterministic preview with operation ID, source/destination qualified identities, expected revisions, requirement map, impacted references and digest. No preview creates a destination spec or changes lifecycle.
- R2: Support transfer to an empty destination and reconciliation against an existing executor. Require explicit per-requirement equivalence, reformulation, withdrawal or pending scope; equal slugs or R-IDs are insufficient.
- R3: Apply the ADR sequence with separate authorization, per-project compare-and-swap and an append-only journal: stage non-executable destination, verify source retirement receipt, activate destination. Never expose two active authorities for one operation.
- R4: Make retry idempotent, detect concurrent writers and crash after every step; resume from verified receipts or compensate safely. A partial operation blocks dependent execution/acceptance and reports the exact recovery step.
- R5: Preserve historical documents/amendments/Git provenance/bundles at source revisions. Resolve old links through a versioned redirect/lineage record with no executable requirements or lifecycle, rejecting redirect cycles and missing targets.
- R6: Rewrite current membership/dependencies only from the approved inventory; changed source refs invalidate the plan. Preserve composition obligations and distinguish prerequisite refs from ownership. Report inaccessible projects as unresolved blockers.
- R7: Keep existing executor reviews bound to their original subjects. A parent adoption commit never impersonates implementation attribution; migrated subjects that change semantics require fresh evidence under the owning authority.
- R8: Provide dry-run/apply/resume diagnostics and contract negotiation in CLI plus authorized read visibility in MCP. Reject old-engine writes to adopted redirect/journal metadata and reject cross-root symlink/traversal escapes.

### Security and compatibility / Segurança e compatibilidade
Treat repository content and references as untrusted. Enforce the existing
project authorization boundary before resolving or disclosing data. Preserve
local-only refs, offline operation and historic sealed contracts. Unsupported
adopted metadata must fail explicitly; no fallback that weakens required evidence.
Bound graph traversal and input size; report stable errors without secrets/roots.

## 3. Technical Plan

### Affected areas / Áreas afetadas
Build a confined migration service over the shared resolver and existing atomic writes. Add versioned journal/redirect schemas and command routing. Freeze the exact command spelling in API docs before implementing; tests invoke the registered CLI, not a standalone migration script.

### Artifacts
- created: .pose/specs/2026-09-21-pose-spec-authority-transfer.md
- created: pose-mcp/internal/pose/spec_transfer.go
- created: pose-mcp/internal/pose/spec_transfer_test.go
- created: pose-mcp/internal/cli/spec_transfer.go
- created: pose-mcp/internal/cli/spec_transfer_test.go
- modified: pose-mcp/internal/pose/spec.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/cli/index.go
- modified: .pose/indexes/validation-matrix.json

Initial exact-path inventory. Reconcile against the implementation revision before
coding and record material changes through amendments. Generated output must be
attributed to its actual producing change; do not reuse this slug for the entire
program. Commit in the owning repository with `POSE-Spec: pose-spec-authority-transfer`.

### Delivery targets
Proposed target: `governance:spec-authority-transfer`, module `pose-mcp`, existing profile `backend-go`, entrypoint `pose-mcp/cmd/pose/main.go`. Register an integration producer executing preview/apply/recovery against two real Git fixtures.
Keep `delivers` empty while this is a draft. Before in-progress, register the
producer, declare the typed target/entrypoint and prove a negative gate case.
Do not count documentation or a test suite matching zero tests as delivery.

### Data and rollout / Dados e rollout
Version new metadata through existing schema/capability mechanisms. Preview
adoption in populated fixtures; preserve old evidence, and reject unknown
contracts on old engines. Rollback must retain audit records and never restore
two executable authorities. No policy activation or migration occurs in planning.

### Risks / Riscos
A partial cross-repository write is unavoidable without a distributed transaction. The journal must make incomplete authority transfer visibly blocked; compensation may never reopen both sides.

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
| Preview, existing executor, history; R1/R2/R5/R7 | `go -C pose-mcp test ./internal/pose ./internal/cli -run TestSpecTransfer -count=1` | Preview is byte-preserving; changed requirements require mapping; historic evidence remains addressable. |
| Crash/replay/concurrent apply; R3/R4/R6 | `go -C pose-mcp test -race ./internal/pose ./internal/cli -run TestSpecTransfer -count=1` | Inject failure at every journal phase; repeated operation creates no duplicate authority. |
| Authorization and capability refusal; R6/R8 | `go -C pose-mcp test ./internal/cli -run TestSpecTransferBoundary -count=1` | Unauthorized writes, stale revisions, redirect loops and symlink escape rejected. |
| Required module matrix; all requirements | `pose validate --strict --module pose-mcp --report` | Registered build/unit/integration checks pass; evidence names this corpus. |

### Governance checks
- `pose lint-spec pose-spec-authority-transfer --ready-check` before activation.
- `pose lint-spec pose-spec-authority-transfer --strict` and `pose check --strict` for structure.
- `pose artifact-check --spec pose-spec-authority-transfer --strict` after correct Git attribution.
- `pose surface-check --spec pose-spec-authority-transfer --strict` after target composition.
- `pose review verify spec:pose-spec-authority-transfer` and `pose closeout-check spec:pose-spec-authority-transfer` before `pose close spec:pose-spec-authority-transfer`.
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
A partial cross-repository write is unavoidable without a distributed transaction. The journal must make incomplete authority transfer visibly blocked; compensation may never reopen both sides.

### Follow-ups
No additional unowned follow-ups; remaining work is in this spec.
