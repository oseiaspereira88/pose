# POSE Report - 2026-08-13

## Report Type
- standard

## Task
- component-aware review provenance
- Task slug: component-aware-review-provenance
- Spec: pose-component-aware-review-plans
- Workflow: review

## Outcome
- Outcome: pass (source: manual)

## Rules Applied
- backend-go
- security
- documentation-style

## Files Changed
- pose/indexes/delivery-integrity.json
- .pose/reports/2026-08-13-standard-component-aware-review-provenance.md
- .pose/reports/2026-08-13-standard-validate-native.md
- .pose/reports/history/standard-validate-native.jsonl
- .pose/results/delivery-validation.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-1b9fd3fd905d
- Selector: range:dbee77a23213b2f4b0b558d5aff474264f86789c..0eb9805d93ec628f196eda4ceffa942c414a6084
- Base: dbee77a23213b2f4b0b558d5aff474264f86789c (dbee77a23213b2f4b0b558d5aff474264f86789c)
- Head: 0eb9805d93ec628f196eda4ceffa942c414a6084 (0eb9805d93ec628f196eda4ceffa942c414a6084)
- Diff digest: sha256:c619eb1c912d934c2c797f786bd8ecd6178ebf07e341bfa3c1779a21d7cc1307
- Paths:
  - created: .pose/adr/2026-08-12-component-aware-effective-review-plans.md
  - created: .pose/changelogs/unreleased/pose-component-aware-review-plans.md
  - created: .pose/knowledge/2026-08-13-decision-log-adr-component-aware-review-plans-review.md
  - created: .pose/knowledge/2026-08-13-handoff-pr15-component-aware-review-provenance.md
  - created: .pose/reports/2026-08-13-standard-component-aware-review-provenance.md
  - created: .pose/reports/history/standard-component-aware-review-provenance.jsonl
  - created: .pose/review-profiles/backend-review.json
  - created: .pose/review-profiles/frontend-review.json
  - created: .pose/reviews/rvw-20260813T032130Z-c42b8c1e.md
  - created: .pose/reviews/rvw-20260813T063956Z-b11e7bcc.md
  - created: pose-mcp/internal/pose/review_plan.go
  - created: pose-mcp/internal/pose/review_plan_test.go
  - created: pose-mcp/internal/scaffold/dist/.pose/review-profiles/backend-review.json
  - created: pose-mcp/internal/scaffold/dist/.pose/review-profiles/frontend-review.json
  - created: pose-mcp/schemas/v1/review-plan.schema.json
  - modified: .agents/skills/pose-review/SKILL.md
  - modified: .pose/assessments/integrations.md
  - modified: .pose/indexes/delivery-integrity.json
  - modified: .pose/indexes/releases.json
  - modified: .pose/indexes/repo-map.json
  - modified: .pose/indexes/spec-graph.json
  - modified: .pose/policy/review.json
  - modified: .pose/reports/2026-08-13-standard-validate-native.md
  - modified: .pose/reports/history/standard-validate-native.jsonl
  - modified: .pose/results/delivery-validation.json
  - modified: .pose/review-profiles/spec-closeout.json
  - modified: .pose/specs/pose-component-aware-review-plans/spec.md
  - modified: .pose/state/integrations.json
  - modified: .pose/workflows/review.md
  - modified: POSE.md
  - modified: docs-site/docs/mcp.md
  - modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
  - modified: locales/pt-BR/.pose/workflows/review.md
  - modified: locales/pt-BR/POSE.md
  - modified: pose-mcp/internal/cli/cli.go
  - modified: pose-mcp/internal/cli/review_closeout.go
  - modified: pose-mcp/internal/cli/review_closeout_test.go
  - modified: pose-mcp/internal/mcpserver/catalog.go
  - modified: pose-mcp/internal/mcpserver/closeout_tool_test.go
  - modified: pose-mcp/internal/mcpserver/server.go
  - modified: pose-mcp/internal/mcpserver/server_test.go
  - modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
  - modified: pose-mcp/internal/pose/review_closeout.go
  - modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/repo-map.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/policy/review.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/review-profiles/spec-closeout.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/review.md
  - modified: pose-mcp/internal/scaffold/dist/POSE.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/review.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
  - modified: pose-mcp/schemas/README.md
  - removed: .pose/changelogs/unreleased/review-legacy-done-scope-exemption.md

## Execution Metadata
- Generated at (UTC): 2026-08-13T07:43:24Z
- Context: review-remediation
- Validation profile: strict
- Sequence for task/spec: 2
- Stable comparison hash: 0059d1ef1d7ae12f0c8992a85f135e21d176d74f54c49cf927a399044057b27f

## Historical Comparison
- Previous execution: 2026-08-13T03:54:35Z
- Status: changed
- Stable field diffs:
- change_set: "" -> "sha256:c619eb1c912d934c2c797f786bd8ecd6178ebf07e341bfa3c1779a21d7cc1307:dbee77a23213b2f4b0b558d5aff474264f86789c:0eb9805d93ec628f196eda4ceffa942c414a6084"
- rules: "" -> "backend-go,security,documentation-style"

## Risks
- high

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
