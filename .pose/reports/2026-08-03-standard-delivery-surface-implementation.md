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
- pose/assessments/integrations.md
- .pose/assessments/technical-debt.md
- .pose/indexes/delivery-integrity.json
- .pose/reports/2026-08-03-standard-delivery-surface-implementation.md
- .pose/reports/history/standard-delivery-surface-implementation.jsonl
- .pose/state/integrations.json
- .pose/state/technical-debt.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- .pose/reports/2026-08-03-standard-delivery-surface-validation.md
- .pose/reports/history/standard-delivery-surface-validation.jsonl
- .pose/results/

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-0fd4bd613f8d
- Selector: range:2f55664..b6b0c10
- Base: 2f55664 (2f556645823c6c1c6c692316cd0116356c31ff59)
- Head: b6b0c10 (b6b0c10afdfc2eae05cc9bdfeae96f124e1557e8)
- Diff digest: sha256:b296ec489b9f786a84be19ac95f62739271a82e98fcfb1fa90f83697aaf2c592
- Paths:
  - created: .agents/skills/pose-surface-closeout/SKILL.md
  - created: .pose/policy/delivery.json
  - created: .pose/reports/2026-08-03-standard-delivery-surface-implementation.md
  - created: .pose/reports/history/standard-delivery-surface-implementation.jsonl
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
  - created: pose-mcp/internal/scaffold/dist/.pose/policy/delivery.json
  - created: pose-mcp/internal/scaffold/dist/.pose/rules/delivery-surface.md
  - created: pose-mcp/internal/scaffold/dist/.pose/workflows/ui-surface.md
  - created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-surface-closeout/SKILL.md
  - created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/delivery-surface.md
  - created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/ui-surface.md
  - modified: .agents/skills/README.md
  - modified: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - modified: .pose/indexes/delivery-integrity.json
  - modified: .pose/indexes/module-metadata.json
  - modified: .pose/indexes/spec-graph.json
  - modified: .pose/indexes/task-map.json
  - modified: .pose/indexes/validation-matrix.json
  - modified: .pose/rules/delivery-evidence.md
  - modified: .pose/specs/pose-delivery-surface-assurance/spec.md
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
  - modified: pose-mcp/internal/pose/spec.go
  - modified: pose-mcp/internal/pose/trace.go
  - modified: pose-mcp/internal/scaffold/dist/.agents/skills/README.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
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
- Generated at (UTC): 2026-08-03T05:17:00Z
- Context: not-provided
- Validation profile: not-provided
- Sequence for task/spec: 3
- Stable comparison hash: 81972a7bf214459d732291e9fceaebd8e8ddca158ba6be5a6f65f040d23b1791

## Historical Comparison
- Previous execution: 2026-08-03T05:12:23Z
- Status: changed
- Stable field diffs:
- change_set: "" -> "sha256:b296ec489b9f786a84be19ac95f62739271a82e98fcfb1fa90f83697aaf2c592:2f556645823c6c1c6c692316cd0116356c31ff59:b6b0c10afdfc2eae05cc9bdfeae96f124e1557e8"
- context: "manual" -> "not-provided"
- rules: "backend-go,security,documentation-style,delivery-evidence,delivery-surface" -> "delivery-evidence,delivery-surface"
- validation_profile: "strict" -> "not-provided"

## Risks
- _No risks provided_

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
