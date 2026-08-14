# POSE Report - 2026-08-13

## Report Type
- standard

## Task
- review bundle convergence
- Task slug: review-bundle-convergence
- Spec: pose-review-bundle-convergence
- Workflow: .pose/workflows/feature.md

## Outcome
- Outcome: pass (source: manual)

## Rules Applied
- .pose/rules/backend-go.md
- .pose/rules/security.md
- .pose/rules/delivery-evidence.md
- .pose/rules/knowledge-governance.md

## Files Changed
- pose/assessments/README.md
- .pose/assessments/consolidated.md
- .pose/assessments/integrations.md
- .pose/assessments/mcp-enforce.md
- .pose/assessments/pose-mcp.md
- .pose/assessments/technical-debt.md
- .pose/indexes/delivery-integrity.json
- .pose/indexes/releases.json
- .pose/indexes/repo-map.json
- .pose/indexes/spec-graph.json
- .pose/reports/2026-08-13-standard-review-bundle-convergence.md
- .pose/reports/history/standard-review-bundle-convergence.jsonl
- .pose/results/delivery-validation.json
- .pose/results/review-bundle-convergence.json
- .pose/state/components/mcp-enforce.json
- .pose/state/components/pose-mcp.json
- .pose/state/integrations.json
- .pose/state/project-state.md
- .pose/state/technical-debt.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/repo-map.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
- .qwen/

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-aecbfc2f5bf1
- Selector: range:1d29b3a..dbd41e3
- Base: 1d29b3a (1d29b3a2b08d093d5401f9deccea74ad4c3723da)
- Head: dbd41e3 (dbd41e3d6cd3f3196f15188525e13d6eb9da9292)
- Diff digest: sha256:13cf29167792a002468acd841fe6972a31acdb166e480fafd503f7b0b583be47
- Paths:
  - created: .pose/adr/2026-08-13-sealed-review-bundles-and-attestations.md
  - created: .pose/changelogs/unreleased/pose-review-bundle-convergence.md
  - created: .pose/knowledge/2026-08-13-decision-log-adr-sealed-review-bundles-review.md
  - created: .pose/reports/2026-08-13-standard-review-bundle-convergence.md
  - created: .pose/reports/history/standard-review-bundle-convergence.jsonl
  - created: .pose/results/review-bundle-convergence.json
  - created: .pose/specs/pose-review-bundle-convergence/spec.md
  - created: pose-mcp/internal/pose/review_bundle.go
  - created: pose-mcp/internal/pose/review_bundle_test.go
  - created: pose-mcp/schemas/v1/review-attestation-envelope.schema.json
  - created: pose-mcp/schemas/v1/review-attestation.schema.json
  - created: pose-mcp/schemas/v1/review-bundle.schema.json
  - created: tests/e2e/review-bundle/run.sh
  - modified: .agents/skills/pose-feature/SKILL.md
  - modified: .agents/skills/pose-review/SKILL.md
  - modified: .pose/assessments/README.md
  - modified: .pose/assessments/consolidated.md
  - modified: .pose/assessments/integrations.md
  - modified: .pose/assessments/pose-mcp.md
  - modified: .pose/assessments/technical-debt.md
  - modified: .pose/indexes/validation-matrix.json
  - modified: .pose/state/components/pose-mcp.json
  - modified: .pose/state/integrations.json
  - modified: .pose/state/technical-debt.json
  - modified: .pose/workflows/feature.md
  - modified: .pose/workflows/review.md
  - modified: POSE.md
  - modified: docs-site/docs/architecture.md
  - modified: docs-site/docs/mcp.md
  - modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
  - modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
  - modified: locales/pt-BR/.pose/workflows/feature.md
  - modified: locales/pt-BR/.pose/workflows/review.md
  - modified: locales/pt-BR/POSE.md
  - modified: pose-mcp/internal/cli/cli.go
  - modified: pose-mcp/internal/cli/cli_test.go
  - modified: pose-mcp/internal/cli/install.go
  - modified: pose-mcp/internal/cli/review_closeout.go
  - modified: pose-mcp/internal/cli/review_closeout_test.go
  - modified: pose-mcp/internal/cli/surface_check.go
  - modified: pose-mcp/internal/cli/usage.go
  - modified: pose-mcp/internal/cli/usage_test.go
  - modified: pose-mcp/internal/cli/validate.go
  - modified: pose-mcp/internal/cli/validate_results.go
  - modified: pose-mcp/internal/mcpserver/catalog.go
  - modified: pose-mcp/internal/mcpserver/closeout_tool_test.go
  - modified: pose-mcp/internal/mcpserver/server.go
  - modified: pose-mcp/internal/mcpserver/server_test.go
  - modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
  - modified: pose-mcp/internal/pose/delivery_surface.go
  - modified: pose-mcp/internal/pose/delivery_surface_test.go
  - modified: pose-mcp/internal/pose/review_closeout.go
  - modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
  - modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/validation-matrix.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/review.md
  - modified: pose-mcp/internal/scaffold/dist/POSE.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/review.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
  - modified: pose-mcp/schemas/README.md

## Execution Metadata
- Generated at (UTC): 2026-08-13T23:59:22Z
- Context: knowledge:project-agnostic-assessment-evidence
- Validation profile: pose-mcp
- Sequence for task/spec: 12
- Stable comparison hash: 95f7382645b89ab96d168a6ff16eddcbbfc5a42de20d0144d834aa58ac9d3acc

## Historical Comparison
- Previous execution: 2026-08-13T23:58:36Z
- Status: changed
- Stable field diffs:
- change_set: "" -> "sha256:13cf29167792a002468acd841fe6972a31acdb166e480fafd503f7b0b583be47:1d29b3a2b08d093d5401f9deccea74ad4c3723da:dbd41e3d6cd3f3196f15188525e13d6eb9da9292"
- context: "auto-validate" -> "knowledge:project-agnostic-assessment-evidence"
- rules: "" -> ".pose/rules/backend-go.md,.pose/rules/security.md,.pose/rules/delivery-evidence.md,.pose/rules/knowledge-governance.md"
- spec: "" -> "pose-review-bundle-convergence"
- validation_profile: "strict" -> "pose-mcp"
- workflow: "" -> ".pose/workflows/feature.md"

## Risks
- high

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
