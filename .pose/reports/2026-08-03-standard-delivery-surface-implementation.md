# POSE Report - 2026-08-03

## Report Type
- standard

## Task
- delivery surface implementation
- Task slug: delivery-surface-implementation
- Spec: pose-delivery-surface-assurance
- Workflow: feature

## Outcome
- Outcome: pass (source: manual)

## Rules Applied
- delivery-evidence
- delivery-surface

## Files Changed
- _No files detected_

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-adb1db3254a1
- Selector: range:2f55664..f62f58a
- Base: 2f55664 (2f556645823c6c1c6c692316cd0116356c31ff59)
- Head: f62f58a (f62f58ad1da381e9c59338cec9f3b6e1116325ea)
- Diff digest: sha256:a35a4ebf7272e12a6d9702c51ebc73d04f33d9cb4f88e8470044b6c8edb5d86f
- Paths:
  - created: .agents/skills/pose-surface-closeout/SKILL.md
  - created: .pose/changelogs/unreleased/pose-delivery-surface-assurance.md
  - created: .pose/policy/delivery.json
  - created: .pose/reports/2026-08-03-pose-delivery-surface-assurance-review.md
  - created: .pose/reports/2026-08-03-standard-delivery-surface-implementation.md
  - created: .pose/reports/2026-08-03-standard-delivery-surface-validation.md
  - created: .pose/reports/history/standard-delivery-surface-implementation.jsonl
  - created: .pose/reports/history/standard-delivery-surface-validation.jsonl
  - created: .pose/results/delivery-validation.json
  - created: .pose/rules/delivery-surface.md
  - created: .pose/workflows/ui-surface.md
  - created: locales/pt-BR/.agents/skills/pose-surface-closeout/SKILL.md
  - created: locales/pt-BR/.pose/rules/delivery-surface.md
  - created: locales/pt-BR/.pose/workflows/ui-surface.md
  - created: pose-mcp/internal/cli/surface_check.go
  - created: pose-mcp/internal/cli/surface_check_test.go
  - created: pose-mcp/internal/mcpserver/surface_assurance_tool_test.go
  - created: pose-mcp/internal/pose/delivery_surface.go
  - created: pose-mcp/internal/pose/delivery_surface_test.go
  - created: pose-mcp/internal/scaffold/dist/.agents/skills/pose-surface-closeout/SKILL.md
  - created: pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-delivery-surface-assurance.md
  - created: pose-mcp/internal/scaffold/dist/.pose/policy/delivery.json
  - created: pose-mcp/internal/scaffold/dist/.pose/results/delivery-validation.json
  - created: pose-mcp/internal/scaffold/dist/.pose/rules/delivery-surface.md
  - created: pose-mcp/internal/scaffold/dist/.pose/workflows/ui-surface.md
  - created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-surface-closeout/SKILL.md
  - created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/delivery-surface.md
  - created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/ui-surface.md
  - modified: .agents/skills/README.md
  - modified: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - modified: .pose/assessments/integrations.md
  - modified: .pose/assessments/technical-debt.md
  - modified: .pose/indexes/delivery-integrity.json
  - modified: .pose/indexes/module-metadata.json
  - modified: .pose/indexes/spec-graph.json
  - modified: .pose/indexes/task-map.json
  - modified: .pose/indexes/validation-matrix.json
  - modified: .pose/rules/delivery-evidence.md
  - modified: .pose/specs/pose-delivery-surface-assurance/spec.md
  - modified: .pose/state/integrations.json
  - modified: .pose/state/technical-debt.json
  - modified: .pose/templates/spec.md
  - modified: .pose/workflows/feature.md
  - modified: POSE.md
  - modified: docs-site/docs/cli.md
  - modified: docs-site/docs/mcp.md
  - modified: locales/pt-BR/.pose/rules/delivery-evidence.md
  - modified: locales/pt-BR/.pose/templates/spec.md
  - modified: locales/pt-BR/.pose/workflows/feature.md
  - modified: locales/pt-BR/POSE.md
  - modified: pose-mcp/internal/cli/artifact_integrity.go
  - modified: pose-mcp/internal/cli/check.go
  - modified: pose-mcp/internal/cli/cli.go
  - modified: pose-mcp/internal/cli/lintspec.go
  - modified: pose-mcp/internal/cli/review_closeout.go
  - modified: pose-mcp/internal/cli/skills_check_test.go
  - modified: pose-mcp/internal/cli/validate.go
  - modified: pose-mcp/internal/cli/validate_results.go
  - modified: pose-mcp/internal/cli/validate_results_test.go
  - modified: pose-mcp/internal/mcpserver/catalog.go
  - modified: pose-mcp/internal/mcpserver/server.go
  - modified: pose-mcp/internal/mcpserver/server_test.go
  - modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
  - modified: pose-mcp/internal/pose/delivery_integrity.go
  - modified: pose-mcp/internal/pose/delivery_integrity_test.go
  - modified: pose-mcp/internal/pose/spec.go
  - modified: pose-mcp/internal/pose/trace.go
  - modified: pose-mcp/internal/scaffold/dist/.agents/skills/README.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/assessments/integrations.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/assessments/technical-debt.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/module-metadata.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/task-map.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/validation-matrix.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/rules/delivery-evidence.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/specs/pose-delivery-surface-assurance/spec.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
  - modified: pose-mcp/internal/scaffold/dist/POSE.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/delivery-evidence.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
  - modified: pose-mcp/internal/scaffold/scaffold.go

## Execution Metadata
- Generated at (UTC): 2026-08-03T05:20:34Z
- Context: not-provided
- Validation profile: not-provided
- Sequence for task/spec: 4
- Stable comparison hash: 30df12a9956680fcc582c4bb0b54f85223f35db7bc915079f9d81aa367feb158

## Historical Comparison
- Previous execution: 2026-08-03T05:17:00Z
- Status: changed
- Stable field diffs:
- change_set: "" -> "sha256:a35a4ebf7272e12a6d9702c51ebc73d04f33d9cb4f88e8470044b6c8edb5d86f:2f556645823c6c1c6c692316cd0116356c31ff59:f62f58ad1da381e9c59338cec9f3b6e1116325ea"

## Risks
- _No risks provided_

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
