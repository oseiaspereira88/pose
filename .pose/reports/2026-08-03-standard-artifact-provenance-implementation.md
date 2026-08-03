# POSE Report - 2026-08-03

## Report Type
- standard

## Task
- artifact provenance implementation
- Task slug: artifact-provenance-implementation
- Spec: pose-artifact-provenance-ledger
- Workflow: feature

## Outcome
- Outcome: pass (source: manual)

## Rules Applied
- backend-go
- security
- documentation-style
- delivery-evidence

## Files Changed
- pose/indexes/delivery-integrity.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- .pose/reports/2026-08-03-standard-artifact-provenance-implementation.md
- .pose/reports/history/standard-artifact-provenance-implementation.jsonl

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-3cdf716fb989
- Selector: range:5403f56..HEAD
- Base: 5403f56 (5403f564d60f0d3f2e7b29b78ab81ac71dbebcc9)
- Head: HEAD (a69466b51c4f323fbd4d197dcd995c67487cd11e)
- Diff digest: sha256:00cafd85a5b9e731872edf7da5906a05a1a5dea641b10aed79487c7195d4f7b3
- Paths:
  - created: .pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - created: .pose/indexes/delivery-integrity.json
  - created: .pose/policy/artifacts.json
  - created: pose-mcp/internal/cli/artifact_integrity.go
  - created: pose-mcp/internal/cli/artifact_integrity_test.go
  - created: pose-mcp/internal/mcpserver/delivery_integrity_tool_test.go
  - created: pose-mcp/internal/pose/delivery_integrity.go
  - created: pose-mcp/internal/pose/delivery_integrity_test.go
  - created: pose-mcp/internal/scaffold/dist/.pose/adr/2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - created: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
  - created: pose-mcp/internal/scaffold/dist/.pose/policy/artifacts.json
  - modified: .pose/indexes/repo-map.json
  - modified: .pose/indexes/roadmaps.json
  - modified: .pose/indexes/services.json
  - modified: .pose/indexes/spec-graph.json
  - modified: .pose/reports/2026-08-02-standard-validate-native.md
  - modified: .pose/reports/history/standard-validate-native.jsonl
  - modified: .pose/rules/delivery-evidence.md
  - modified: .pose/specs/pose-artifact-provenance-ledger/spec.md
  - modified: .pose/templates/spec.md
  - modified: .pose/workflows/feature.md
  - modified: POSE.md
  - modified: docs-site/docs/mcp.md
  - modified: locales/pt-BR/.pose/templates/spec.md
  - modified: locales/pt-BR/POSE.md
  - modified: pose-mcp/internal/cli/check.go
  - modified: pose-mcp/internal/cli/cli.go
  - modified: pose-mcp/internal/cli/index.go
  - modified: pose-mcp/internal/cli/lintspec.go
  - modified: pose-mcp/internal/cli/report.go
  - modified: pose-mcp/internal/cli/review_closeout.go
  - modified: pose-mcp/internal/mcpserver/catalog.go
  - modified: pose-mcp/internal/mcpserver/server.go
  - modified: pose-mcp/internal/mcpserver/server_test.go
  - modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/repo-map.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/roadmaps.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/services.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/rules/delivery-evidence.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/specs/pose-artifact-provenance-ledger/spec.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
  - modified: pose-mcp/internal/scaffold/dist/POSE.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

## Execution Metadata
- Generated at (UTC): 2026-08-03T04:01:34Z
- Context: manual
- Validation profile: strict
- Sequence for task/spec: 2
- Stable comparison hash: e3b2a23931677b6a9cecb6bf77f192c29853a86667eb0bf9fe34721bbc3c5320

## Historical Comparison
- Previous execution: 2026-08-03T03:59:12Z
- Status: changed
- Stable field diffs:
- change_set: "" -> "sha256:00cafd85a5b9e731872edf7da5906a05a1a5dea641b10aed79487c7195d4f7b3:5403f564d60f0d3f2e7b29b78ab81ac71dbebcc9:a69466b51c4f323fbd4d197dcd995c67487cd11e"

## Risks
- _No risks provided_

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
