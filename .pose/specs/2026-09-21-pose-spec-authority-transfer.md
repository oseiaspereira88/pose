---
slug: pose-spec-authority-transfer
status: done
created_at: 2026-09-21
completed_at: 2026-09-24
supersedes:
depends_on: pose-qualified-artifact-resolution
priority: 0
components: pose-mcp
task_type: feature
delivers: governance:spec-authority-transfer
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
Build a confined migration service over the existing authorized project roots and shared artifact resolver. Freeze the CLI contract as `pose spec-transfer preview|apply|resume|status`; preview emits a deterministic plan, apply requires its digest and explicit authorization for both project IDs, and resume accepts only a verified operation receipt. MCP exposes read-only status for one already-authorized project. No command commits or pushes.

Use review-policy schema 4 with `spec_authority_transfer_version: 1` as the opt-in capability boundary. Do not stamp it during `pose update` or enable it in consumer projects. A transfer must fail closed unless both roots explicitly adopted the capability.

### Artifacts
- modified: .pose/specs/2026-09-21-pose-spec-authority-transfer.md
- created: pose-mcp/internal/pose/spec_transfer.go
- created: pose-mcp/internal/pose/spec_transfer_test.go
- created: pose-mcp/internal/pose/spec_transfer_locks.go
- created: pose-mcp/internal/pose/spec_transfer_lock_unix.go
- created: pose-mcp/internal/pose/spec_transfer_lock_windows.go
- created: pose-mcp/internal/pose/spec_transfer_lock_other.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/pose/artifact_ref_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- created: pose-mcp/internal/cli/spec_transfer.go
- created: pose-mcp/internal/cli/spec_transfer_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- created: pose-mcp/internal/mcpserver/spec_transfer_test.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json

Initial exact-path inventory. Reconcile against the implementation revision before
coding and record material changes through amendments. Generated output must be
attributed to its actual producing change; do not reuse this slug for the entire
program. Commit in the owning repository with `POSE-Spec: pose-spec-authority-transfer`.

### Delivery targets
- governance:spec-authority-transfer module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

The target is now declared because the CLI/MCP producers and the negative gate
have executed against independent Git fixtures. Keep this scope in progress
until the complete module matrix, attribution, review and closeout gates pass.
Do not count documentation or a test suite matching zero tests as delivery.

### Data and rollout / Dados e rollout
Version new metadata through existing schema/capability mechanisms. Preview
adoption in populated fixtures; preserve old evidence, and reject unknown
contracts on old engines. Rollback must retain audit records and never restore
two executable authorities. No policy activation or migration occurs in planning.

### Risks / Riscos
A partial cross-repository write is unavoidable without a distributed transaction. The journal must make incomplete authority transfer visibly blocked; compensation may never reopen both sides.

## 4. Tasks

- [x] Confirm source revisions, accepted ADR and local prerequisites; resolution is the completed predecessor.
- [x] Reconcile the proposed artifact inventory, typed target and test plan before implementation.
- [x] Implement the requirements incrementally with regression/contract tests.
- [x] Update public contracts, consumers, docs/locales and generated scaffold where affected.
- [x] Run the scenarios below plus required module checks and record evidence per R-ID.
- [x] Obtain explicit review and governed closeout; disposition follow-ups and refresh assessments.

## 5. Decisions

- Date: 2026-09-24. Status: Adopted for implementation.
- Adopt [qualified authority and transfer](../adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md) for this scope; its status is Accepted.
- Prefer the existing Store/roots/review contracts over a second authority or resolver.
- Preserve the distinction between source implementation, consumer adoption and program acceptance.
- Reuse `knowledge:contract-baseline-handoff` for the existing multi-project authorization boundary; no new project registry is introduced.

## 6. Validation

### Strategy / Estratégia
Risk: high for authority and cross-project acceptance. The named new test families
and scripts below are implementation deliverables, not checks already executed.
All rows are mandatory. Verify that each new named family exists and actually
executes; a successful command with no matching tests is insufficient evidence.
Use independent Git fixtures, including nested submodule and sibling layouts.

| Scenario / requisitos | Command (from this repository root) | Expected evidence |
| --- | --- | --- |
| Preview, requirement mapping, history, frontmatter, folder companion files and CLI; R1/R2/R5/R7 | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'TestSpecTransfer(Preview|RequirementMap|ApplyPreservesHistory|ApplyPreservesSourceFrontmatter|PreservesFolder|CLIEndToEnd)' -count=1` | Preview is read-only and stable; every requirement is accounted for; transferred metadata and old evidence stay intact. |
| Crash/replay/concurrent apply; R3/R4/R6 | `go -C pose-mcp test -race ./internal/pose ./internal/cli -run 'TestSpecTransfer(Apply|Resume|Concurrent|Interrupted|CLIEndToEnd)' -count=1` | Injected interruption at each phase resumes from verified receipts; duplicate or concurrent authority is rejected. |
| Authorization, stale revisions, changed reference inventory, redirect cycles/missing targets, capability and confinement; R5/R6/R8 | `go -C pose-mcp test ./internal/pose ./internal/cli -run 'TestSpecTransfer(Negative|PreviewBlocks|ResumeBlocks|ResolverRejects|ApplyPreservesHistory|PreservesFolder|RewritesRoadmap)|TestSpecAuthorityTransferPolicySchema' -count=1` | Unauthorized roots, CAS conflicts, changed source-reference inventory, invalid redirect chains, unsupported policy and symlink/traversal escapes fail closed. |
| Authorized read visibility and project scoping in MCP; R8 | `go -C pose-mcp test ./internal/mcpserver -run TestSpecTransferStatus -count=1` | Only the selected authorized project's path-free operation status is returned. |
| Required module matrix; all requirements | `/tmp/pose-harne8-compatible validate --strict --module pose-mcp --report` | Registered build/unit/integration checks pass; evidence names this corpus. |

### Governance checks
- `pose lint-spec pose-spec-authority-transfer --ready-check` before activation.
- `pose lint-spec pose-spec-authority-transfer --strict` and `pose check --strict` for structure.
- `pose artifact-check --spec pose-spec-authority-transfer --strict` after correct Git attribution.
- `pose surface-check --spec pose-spec-authority-transfer --strict` after target composition.
- `pose review verify spec:pose-spec-authority-transfer` and `pose closeout-check spec:pose-spec-authority-transfer` before `pose close spec:pose-spec-authority-transfer`.
- Run `pose assess discover --update-state` at implementation closeout, not to mark this plan delivered.

### Execution log / Log de execução
2026-09-24: accepted ADR and predecessor confirmed; risk-based test plan and exact artifact inventory recorded. Implemented the transfer engine, CLI, resolver redirect and MCP status. Preview/apply/recovery, negative-gate, redirect-cycle, historical-subject and authorized-status test families pass on Git fixtures. Follow-up hardening preserves unknown source frontmatter and blocks resume when new source references appear after an interruption. `go test ./...`, `go vet ./...`, all four named scenario commands, and the strict module matrix pass (21/21 checks). `pose check --strict` and `pose lint-spec --strict` pass; check reports 11 existing non-blocking warnings. `pose artifact-check --strict` passes with existing repository-wide orphan warnings; `pose surface-check --strict` passes (1 target, 21 results, 0 findings). `pose docs-check` reports no docs manifest, so the opt-in docs contract is absent; `pose assess integrate` reports 56 unobserved provider contracts, including the new status tool. Human review and governed closeout remain pending.

2026-09-24 autonomous review: sealed bundle `rvb-165fc70dcb37c601`
and attestation `rva-c5dc094230af32dc` verified as approved and fresh.
`pose close` completed the spec; the earlier pending-review note records the
state before this review.

### Requirement trace
- R1 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferPreviewIsDeterministicAndReadOnly — repeated previews have the same digest and leave both Git roots unchanged; sibling and nested-submodule roots are covered.
- R2 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferRequirementMapRequiresExplicitCoverage — every requirement needs an explicit disposition and valid destination mapping.
- R3 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferInterruptedOperationsResumeWithoutDuplicateAuthority — the staged journal resumes through activation without two executable authorities.
- R4 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferInterruptedOperationsResumeWithoutDuplicateAuthority test:TestSpecTransferConcurrentApplySerializesWriters — interrupted retries and competing writers preserve one serialized operation.
- R5 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferApplyPreservesHistoryAndRewritesApprovedReferences test:TestSpecTransferResolverRejectsRedirectCycles — historical artifacts stay at source and invalid redirect chains fail closed.
- R6 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferRewritesRoadmapMembershipAndKeepsOtherObligations test:TestSpecTransferPreviewBlocksUnscannedProjects test:TestSpecTransferResumeBlocksChangedReferenceInventory — only inventoried ownership references change; unresolved roots and newly introduced source references block the operation.
- R7 [satisfied] check:spec-authority-transfer-integration test:TestSpecTransferApplyPreservesHistoryAndRewritesApprovedReferences — the source history and existing review subjects remain bound to their original revisions.
- R8 [satisfied] check:spec-authority-transfer-integration check:spec-authority-transfer-negative-gates check:spec-authority-transfer-mcp-status test:TestSpecTransferCLIEndToEndAndAuthorization test:TestSpecTransferStatusAuthorization — CLI authorization and MCP project-scoped status are covered by the passing module matrix.

## 7. Final Report

### Delivered scope / Escopo entregue
The transfer engine, CLI and MCP status contract passed review and governed spec closeout. No consumer project has adopted schema 4 and no real project transfer has been executed.

### Residual risks / Riscos residuais
A partial cross-repository write is unavoidable without a distributed transaction. The journal must make incomplete authority transfer visibly blocked; compensation may never reopen both sides.

### Follow-ups
No additional unowned follow-ups; remaining work is in this spec.
