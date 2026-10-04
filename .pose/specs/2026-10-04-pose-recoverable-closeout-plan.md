---
slug: pose-recoverable-closeout-plan
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-typed-producer-diagnostics, pose-obligation-projection, pose-phase-scoped-readiness
priority: 1
components: pose-mcp
delivers: surface:recoverable-closeout-plan
task_type: feature
---

# Spec: Evolve closeout into a recoverable mechanical plan

## 1. Intent

### Goal

Let the existing closeout machinery compute and apply a checkpointed plan — verify
context, regenerate stale evidence at `results_path`, index, seal, prepare mechanical
criteria, expose pending judgment, re-verify and transition — resumable after
interruption.

### Business value

The skill makes the operator execute an order-sensitive choreography and documents
bundles sealing stale results because `results_path` was not regenerated.

### Constraints

Recoverable operation with checkpoints and idempotent reapplication; no ACID promise
over Git and external services. Judgment is never filled.

Program source: backlog items POSE-21 (P1, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F09; sources
E07, E08, E13). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Publishing on spec close; auto-approving review or editing policy.

### Anti-mechanization guardrail

Operação recuperável, sem prometer transação ACID sobre Git e serviços externos.

## 2. Requirements

### Functional

- R1: `pose close spec:<slug> --plan` shall preview the ordered steps with reasons and the pending judgments; `--apply` executes mechanical steps; `--resume` continues from the last checkpoint.
- R2: `results_path` shall be regenerated when stale before sealing the corresponding result.
- R3: Interrupting after prepare, index or seal shall allow resumption without duplicate bundles or attestations.
- R4: A material change between plan and apply shall require revalidation or a new plan.
- R5: The plan shall stop with pending judgment obligations when no conclusion exists and never write a judgment answer.
- R6: The closeout skill shall be shortened to the plan flow, keeping the primitive commands documented for diagnosis.

### Non-functional

- Checkpoint file bounded and removed on completion.

### Security

- Plan apply revalidates context and authority.

### Compatibility

- Primitive commands unchanged.

## 3. Technical Plan

### Affected areas

Closeout domain, continuous closeout, CLI/MCP close, closeout skill.

### Artifacts

- created: .pose/specs/2026-10-04-pose-recoverable-closeout-plan.md
- created: pose-mcp/internal/pose/closeout_plan.go
- created: pose-mcp/internal/cli/closeout_plan.go
- created: pose-mcp/internal/cli/closeout_plan_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/adversarial_corpus_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-recoverable-closeout-plan.md

Reconciled against the tree at activation.

### Delivery targets

- surface:recoverable-closeout-plan module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Hidden automation surprising operators; preview first, explicit apply.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-recoverable-closeout-plan`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Crash-injection tests at each checkpoint; stale results_path regression; judgment-never-
filled corpus case.

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
- Command: `pose lint-spec pose-recoverable-closeout-plan --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:recoverable-closeout-plan evidence:integration test:TestClosePlanPreviewsWithoutWriting check:closeout-plan-integration test:TestAnInterruptedCloseoutResumesWithoutDuplication
- R2 [satisfied] <evidence is regenerated into results_path and indexed before sealing> test:TestAnInterruptedCloseoutResumesWithoutDuplication check:closeout-plan-integration
- R3 [satisfied] test:TestAnInterruptedCloseoutResumesWithoutDuplication check:closeout-plan-integration
- R4 [satisfied] test:TestAStalePlanDigestIsRefused check:closeout-plan-integration
- R5 [satisfied] test:TestTheClosePlanStopsAtJudgmentAndNeverAnswersIt check:closeout-plan-integration
- R6 [satisfied] <the pose-spec-closeout skill leads with the plan flow and keeps the primitive commands for diagnosis>

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
