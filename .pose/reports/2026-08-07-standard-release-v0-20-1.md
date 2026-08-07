# POSE Report - 2026-08-07

## Report Type
- standard

## Task
- release-v0.20.1
- Task slug: release-v0-20-1

## Outcome
- Outcome: pass (source: derived)

## Rules Applied
- _Not provided_

## Files Changed
- pose/changelogs/unreleased/pose-extension-signing-dirty-tree.md
- .pose/indexes/delivery-integrity.json
- .pose/indexes/releases.json
- README.md
- compatibility.json
- docs-site/docs/ci.md
- pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-extension-signing-dirty-tree.md
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- pose-mcp/internal/scaffold/dist/README.md
- pose-mcp/internal/version/version.go
- pose-mcp/server.json
- .pose/changelogs/v0.20.1.md
- .pose/changelogs/v0.20.1/
- .pose/releases/v0.20.1/
- .pose/reports/pose-validate.latest.log
- pose-mcp/internal/scaffold/dist/.pose/changelogs/v0.20.1.md
- pose-mcp/internal/scaffold/dist/.pose/changelogs/v0.20.1/
- pose-mcp/internal/scaffold/dist/.pose/releases/v0.20.1/

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

## Execution Metadata
- Generated at (UTC): 2026-08-07T19:05:52Z
- Context: release
- Validation profile: strict
- Sequence for task/spec: 1
- Stable comparison hash: 1cddae77fc6719f6c7f42946c067d112c0c54c37b82ba648e9f98702df8cd0dd

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
