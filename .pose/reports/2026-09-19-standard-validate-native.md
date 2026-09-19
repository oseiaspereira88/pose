# POSE Report - 2026-09-19

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
- pose/assessments/integrations.md
- .pose/indexes/delivery-integrity.json
- .pose/indexes/spec-graph.json
- .pose/reports/2026-09-19-standard-validate-native.md
- .pose/reports/history/standard-validate-native.jsonl
- .pose/results/delivery-validation.json
- .pose/state/integrations.json

## Validation Commands
- go build ./...
- go test ./...
- go vet ./...
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run Delivery|Surface|RoadmapCheck|Contributor -count=1
- go test ./internal/cli ./internal/mcpserver -run Surface|DeliveryIntegrity|RoadmapCheck|Contributor -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run ReviewBundle|ReviewAttestation|ReviewPlanGroupsRepeatedWarnings|ReviewPlanActionableToolPhases|ToolCatalog -count=1

## Results
- - [pass] pose-mcp/go/build (0.8s)
- - [pass] pose-mcp/go/test (17.0s)
- - [pass] pose-mcp/go/vet (2.3s)
- - [pass] pose-mcp/go/delivery-integration (0.9s)
- - [pass] pose-mcp/go/delivery-reachability (0.7s)
- - [pass] pose-mcp/go/review-bundle-convergence (1.0s)
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-09-19T04:07:28Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 129
- Stable comparison hash: 5b47855e60f64e73728abd99582eb01357a94f0c289ad7fa9125d680a322e54f

## Historical Comparison
- Previous execution: 2026-09-19T04:02:58Z
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
