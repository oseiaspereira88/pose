# POSE Report - 2026-09-11

## Report Type
- standard

## Task
- validate-native
- Task slug: validate-native

## Outcome
- Outcome: pass (source: derived)

## Rules Applied
- _Not provided_

## Files Changed
- pose/changelogs/unreleased/pose-manual-merge-backs-up-only-local-edits.md
- .pose/specs/2026-09-11-pose-manual-merge-backs-up-only-local-edits.md
- README.md
- README.pt-BR.md
- compatibility.json
- docs-site/docs/ci.md
- pose-mcp/internal/version/version.go
- pose-mcp/server.json
- .pose/changelogs/v5.0.5.md
- .pose/changelogs/v5.0.5/
- .pose/releases/v5.0.5/

## Validation Commands
- go build ./...
- go test ./...
- go vet ./...
- go build ./...
- go test ./...
- go vet ./...
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run Delivery|Surface|RoadmapCheck|Contributor -count=1
- go test ./internal/cli ./internal/mcpserver -run Surface|DeliveryIntegrity|RoadmapCheck|Contributor -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run ReviewBundle|ReviewAttestation|ReviewPlanGroupsRepeatedWarnings|ReviewPlanActionableToolPhases|ToolCatalog -count=1

## Results
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-09-11T18:12:58Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 118
- Stable comparison hash: 5b47855e60f64e73728abd99582eb01357a94f0c289ad7fa9125d680a322e54f

## Historical Comparison
- Previous execution: 2026-09-11T16:42:23Z
- Status: stable
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
