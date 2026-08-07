# POSE Report - 2026-08-07

## Report Type
- standard

## Task
- closeout-pose-compat-gate-manual-refresh-assertion
- Task slug: closeout-pose-compat-gate-manual-refresh-assertion
- Spec: pose-compat-gate-manual-refresh-assertion

## Outcome
- Outcome: pass (source: derived)

## Rules Applied
- _Not provided_

## Files Changed
- .pose/reports/pose-validate.latest.log

## Validation Commands
- go test ./...
- go vet ./...
- go test ./...
- go vet ./...
- go test ./...
- go vet ./...
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run Delivery|Surface|RoadmapCheck -count=1
- go test ./internal/cli ./internal/mcpserver -run Surface|DeliveryIntegrity|RoadmapCheck -count=1

## Results
- Result: SUCCESS

## Change Set
- ID: cs-5a03d8ddd0af
- Selector: range:d9ee137..HEAD
- Base: d9ee137 (d9ee1375c6e54f6fc365528faf573264fc389d63)
- Head: HEAD (4c6a8f2e407c48aa1b89dca406c5485d1add2100)
- Diff digest: sha256:12c79de6c57acc8eec647c722bdbd047b34edb835d354ea4a0f99a54ea5a8291
- Paths:
  - created: .pose/changelogs/v0.20.0.md
  - created: .pose/changelogs/v0.20.0/pose-compat-gate-manual-refresh-assertion.md
  - created: .pose/releases/v0.20.0/manifest.json
  - created: .pose/specs/pose-compat-gate-manual-refresh-assertion/spec.md
  - created: pose-mcp/internal/scaffold/dist/.pose/changelogs/v0.20.0.md
  - created: pose-mcp/internal/scaffold/dist/.pose/changelogs/v0.20.0/pose-compat-gate-manual-refresh-assertion.md
  - created: pose-mcp/internal/scaffold/dist/.pose/releases/v0.20.0/manifest.json
  - created: pose-mcp/internal/scaffold/dist/.pose/specs/pose-compat-gate-manual-refresh-assertion/spec.md
  - modified: .pose/indexes/delivery-integrity.json
  - modified: .pose/indexes/releases.json
  - modified: .pose/indexes/spec-graph.json
  - modified: .pose/state/history.jsonl
  - modified: .pose/state/project-state.md
  - modified: .pose/state/refresh-log.jsonl
  - modified: README.md
  - modified: compatibility.json
  - modified: docs-site/docs/ci.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
  - modified: pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
  - modified: pose-mcp/internal/scaffold/dist/README.md
  - modified: pose-mcp/internal/version/version.go
  - modified: pose-mcp/server.json
  - modified: tests/release/compat.sh
  - renamed: .pose/changelogs/unreleased/pose-extension-reference-publication.md -> .pose/changelogs/v0.20.0/pose-extension-reference-publication.md
  - renamed: pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-extension-reference-publication.md -> pose-mcp/internal/scaffold/dist/.pose/changelogs/v0.20.0/pose-extension-reference-publication.md

## Execution Metadata
- Generated at (UTC): 2026-08-07T18:59:24Z
- Context: closeout
- Validation profile: strict
- Sequence for task/spec: 1
- Stable comparison hash: f493de7e5af93ade69db20319089bf7dff8252b07cb3ed93e29118071608cf8e

## Historical Comparison
- Previous execution: _No previous execution_
- Status: first-run
- Stable field diffs:
- _No changes in stable fields_

## Risks
- _No risks provided_

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
