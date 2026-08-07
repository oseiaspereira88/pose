# POSE Report - 2026-08-07

## Report Type
- standard

## Task
- release-v0.20.3
- Task slug: release-v0-20-3

## Outcome
- Outcome: pass (source: derived)

## Rules Applied
- _Not provided_

## Files Changed
- pose/indexes/delivery-integrity.json
- .pose/reports/2026-08-07-standard-validate-native.md
- .pose/reports/history/standard-validate-native.jsonl
- .pose/results/delivery-validation.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- pose-mcp/internal/scaffold/dist/.pose/results/delivery-validation.json
- .pose/reports/2026-08-07-standard-closeout-pose-verifier-extension-install-cwd.md
- .pose/reports/history/standard-closeout-pose-verifier-extension-install-cwd.jsonl
- .pose/reports/pose-validate.latest.log
- .pose/reviews/rvw-20260807T193706Z-182a5b45.md
- pose-mcp/internal/scaffold/dist/.pose/reviews/rvw-20260807T193706Z-182a5b45.md

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
- Generated at (UTC): 2026-08-07T19:37:31Z
- Context: release
- Validation profile: strict
- Sequence for task/spec: 1
- Stable comparison hash: 64ef7e70df80e2a5f6f4cb87b3ad7b1c4d468a25100b08aef3c9ebc33f8892d9

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
