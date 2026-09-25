---
slug: pose-agent-project-context
status: in-progress
created_at: 2026-09-21
completed_at:
supersedes:
depends_on: pose-qualified-artifact-resolution, pose-federated-roadmap-acceptance, pose-spec-authority-transfer
priority: 0
components: pose-mcp
task_type: feature
delivers: surface:multirepo-agent-context
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
- R8: Keep single-repository UX compatible. Publish capabilities and stable errors; expose a distinct qualified-create verb that old engines reject before writing, and add installed-distribution tests proving old engines cannot apply new transfer/federation semantics.

### Security and compatibility / Segurança e compatibilidade
Treat repository content and references as untrusted. Enforce the existing
project authorization boundary before resolving or disclosing data. Preserve
local-only refs, offline operation and historic sealed contracts. Unsupported
adopted metadata must fail explicitly; no fallback that weakens required evidence.
Bound graph traversal and input size; report stable errors without secrets/roots.

## 3. Technical Plan

### Affected areas / Áreas afetadas
Add a read-only `pose context` projection over the existing project resolver and
extend `pose_mcp_context` with the same qualified-task result. Bind each context
to the selected logical project, canonical artifact, observed Git revision,
artifact digest, configured checkout identity and bounded adopted-policy
digests, including uncommitted review-policy edits. Reuse `ArtifactResolver` for local refs, xrefs and redirects;
fail closed on unknown, competing, incomplete-transfer, stale or unauthorized
targets. Require an explicit `POSE_PROJECT_ROOTS` binding plus the matching
context digest for CLI cross-project writes. Keep closeout evaluation and writes
inside the authority project. Update the distributed instruction sources and
regenerate their embedded scaffold with the existing generator.
Use `new-spec-qualified` for cross-project creation. It requires the qualified
task and context digest and delegates to the existing resolver; the older engine
rejects the unfamiliar verb before its permissive `new-spec` parser can write.

### Artifacts
- modified: .pose/specs/2026-09-21-pose-agent-project-context.md
- modified: .pose/adr/2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/help_test.go
- created: pose-mcp/internal/cli/project_context.go
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/pose/agent_project_context.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/mcp_context_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/cli/multirepo_agent_flow_test.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: .pose/workflows/feature.md
- modified: locales/pt-BR/.pose/workflows/feature.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: .pose/indexes/validation-matrix.json

Initial exact-path inventory. Reconcile against the implementation revision before
coding and record material changes through amendments. Generated output must be
attributed to its actual producing change; do not reuse this slug for the entire
program. Commit in the owning repository with `POSE-Spec: pose-agent-project-context`.

### Delivery targets
- surface:multirepo-agent-context module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

Register `multi-repo-agent-context-reachability` (`reachability`),
`multi-repo-agent-context-integration` (`integration`),
`multi-repo-agent-negative-gates` (`integration`) and
`multi-repo-agent-installed-journey` (`e2e`) in the `pose-mcp` validation
matrix. Each producer must name a test family that exists and executes. The
negative producer must cover ambiguity, stale context, denied project access,
redirect/transfer blocks and old-contract refusal.

### Data and rollout / Dados e rollout
Version new metadata through existing schema/capability mechanisms. Preview
adoption in populated fixtures; preserve old evidence, and reject unknown
contracts on old engines. Rollback must retain audit records and never restore
two executable authorities. No policy activation or migration occurs in planning.

### Risks / Riscos
Hidden default project selection is the recurrence trigger. Explicit qualified intent must survive cwd changes; a same-slug lookup alone must not imply user intent or write authorization.

## 4. Tasks

- [x] Confirm source revisions, accepted ADRs and local prerequisites; dependent implementation exists, while separate review/closeout remains pending.
- [x] Reconcile artifacts and register target, four evidence producers and named negative-gate family.
- [x] Implement the requirements incrementally with regression/contract tests.
- [x] Update public contracts, consumers, docs/locales and generated scaffold where affected.
- [x] Run the scenarios below plus required module checks and record evidence per R-ID.
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
| Context, ambiguity, duplicate prevention; R1–R4 | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgent -count=1` | Parent, child and sibling roots resolve the same qualified task; no local shadow is created; stale or unauthorized writes fail before mutation. |
| Negative gates; R2–R4, R8 | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentNegative -count=1` | Ambiguous task, stale revision, denied project, redirect/transfer conflict and unsupported adopted contract fail closed. |
| Installed journey, contract mismatch; R6–R8 | `go -C pose-mcp test ./internal/cli -run TestMultiRepoAgentInstalled -count=1` | Built CLI recomputes context after target checkout changes, blocks stale writes, and creates in the explicit authority after a fresh context. |
| Old installed engine guard; R8 (required) | `bash ../tests/e2e/pose-multirepo/run.sh adoption` | The consumer's required context-first path stops before `new-spec` when the pinned engine lacks `context`; the direct legacy probe remains an adoption blocker until the pin is replaced. |
| Reachability; R1/R8 | `go -C pose-mcp test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentSurface -count=1` | `pose context` and `pose_mcp_context` expose the same path-free qualified task contract. |
| Instruction/scaffold parity; R5/R8 | `go -C pose-mcp test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1` | Regenerated locales/skills match canonical sources; new flow documented. |
| Required module matrix; all requirements | `pose validate --strict --module pose-mcp --json .pose/results/delivery-validation.json --report` | Registered build/unit/integration checks pass; structured evidence includes this corpus. |

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
2026-09-24: reconciled the target, exact files and dedicated reachability,
integration, negative-gate and installed-journey producers before implementation.
Read-only discovery will expose a context digest that mutations must match;
cross-project writes require an explicit POSE_PROJECT_ROOTS binding.
2026-09-24: all named CLI/MCP families, scaffold parity, full `go test ./...`,
and `go vet ./...` passed. The strict module matrix passed 25/25 checks.
`pose lint-spec pose-agent-project-context --strict` passed with two
in-progress dependency warnings. `pose artifact-check --strict` observed the
same 35 declared and changed paths, with 385 repository-wide orphan warnings.
`pose surface-check --spec pose-agent-project-context --strict` passed with
zero findings.
`pose assess integrate` ran and reported 56 generic unobserved-provider gaps.
`pose check --strict` now passes with 11 pre-existing warnings when run with the
compatible candidate engine. `docs-check` has no configured docs manifest.
`govulncheck` could not load Go 1.27 source packages because the available
scanner was built with Go 1.26.
The Harne8 adoption fixture built the old gitlink binary and found that it
ignores `new-spec --task xref:...`, creating a local draft. Autonomous review
`rva-b178e83cb461ba31` requested changes; the consumer remains inactive.
The consumer's context-first fixture was rerun on 2026-09-24 and stopped the
old engine before a qualified write. A direct call to that immutable binary
still creates a local draft. The old engine also rejects the new transfer
command and adopted review policy, so it cannot apply transfer or federation
semantics. The consumer's inactive binding and pin/version gate remain required
until it installs this contract; the source result does not authorize adoption.
On 2026-09-24 (America/Recife), the candidate `pose-mcp` matrix passed 25/25
outside the sandbox, where MCP tests could open loopback sockets.
The compatibility review found that a context-first convention alone did not
make the old `new-spec` invocation safe. The distinct `new-spec-qualified`
command and its old-engine rejection fixture are the follow-up implementation;
the previous R8 disposition must be revalidated on the new source revision.

### Requirement trace
- R1 [satisfied] test:TestMultiRepoAgentSurfaceExposesPathFreeCLIContext test:TestMultiRepoAgentSurfaceReportsPathFreeQualifiedTaskContext
- R2 [satisfied] test:TestMultiRepoAgentContextResolvesOneQualifiedTaskFromEveryCheckout test:TestMultiRepoAgentRoutingClosesOnlyQualifiedAuthorityWithFreshContext
- R3 [satisfied] test:TestMultiRepoAgentRoutingCreatesAtQualifiedAuthorityAndReusesIt test:TestMultiRepoAgentNegativeTransferBlocksUntilCanonicalAuthorityIsActive
- R4 [satisfied] test:TestMultiRepoAgentNegativeBindingChangeInvalidatesContext test:TestMultiRepoAgentNegativePolicyChangeInvalidatesContext test:TestMultiRepoAgentNegativeContextDeniesUnauthorizedAuthorityAndUnknownContract
- R5 [satisfied] test:TestEmbeddedDistMatchesPoseDist test:TestSkillLocaleParity
- R6 [satisfied] test:TestMultiRepoAgentInstalledJourneyUsesInstalledCLIAndRejectsStaleBinding test:TestMultiRepoAgentNegativeBindingChangeRequiresFreshMCPConnection
- R7 [satisfied] test:TestMultiRepoAgentRoutingClosesOnlyQualifiedAuthorityWithFreshContext
- R8 [deferred-integration: the distinct qualified-create verb is implemented but its installed old-engine rejection and full matrix must pass on the new revision] test:TestMultiRepoAgentInstalledJourneyUsesInstalledCLIAndRejectsStaleBinding test:TestMultiRepoAgentNegativeContextDeniesUnauthorizedAuthorityAndUnknownContract e2e:harne8/multirepo-adoption-preflight

## 7. Final Report

### Delivered scope / Escopo entregue
Path-free project/task context is shared by CLI and MCP; qualified create, review
and closeout routes to the explicit canonical authority with stale-context gates.
The instruction sources, pt-BR locale and embedded scaffold now describe that
flow. Requirements R1–R7 have named passing coverage; R8 requires the new
installed compatibility run and renewed review.
The old-engine fixture still shows that direct invocation can create a local
draft; Harne8 must keep its consumer binding inactive and replace the pinned
engine before allowing qualified operations.

### Residual risks / Riscos residuais
Hidden default project selection is the recurrence trigger. Explicit qualified intent must survive cwd changes; a same-slug lookup alone must not imply user intent or write authorization.

### Follow-ups
No additional unowned follow-ups; remaining work is in this spec.
