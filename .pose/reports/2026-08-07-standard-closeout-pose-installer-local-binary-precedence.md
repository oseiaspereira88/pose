# POSE Report - 2026-08-07

## Report Type
- standard

## Task
- closeout-pose-installer-local-binary-precedence
- Task slug: closeout-pose-installer-local-binary-precedence
- Spec: pose-installer-local-binary-precedence

## Outcome
- Outcome: fail (source: derived)

## Rules Applied
- _Not provided_

## Files Changed
- pose/indexes/delivery-integrity.json
- .pose/indexes/releases.json
- .pose/indexes/spec-graph.json
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
- Result: FAILURE (required check failed)

## Change Set
- ID: cs-211d56f0088f
- Selector: range:76857a998f48c72c3ef057cb82b87a55afd89c75..964086bd9b152d560e7e681941676d77b5b1293f
- Base: 76857a998f48c72c3ef057cb82b87a55afd89c75 (76857a998f48c72c3ef057cb82b87a55afd89c75)
- Head: 964086bd9b152d560e7e681941676d77b5b1293f (964086bd9b152d560e7e681941676d77b5b1293f)
- Diff digest: sha256:71a94a7e2cb664533be14725f310925b680f1740c4315b562807f6215d511dcb
- Paths:
  - created: .pose/changelogs/unreleased/pose-installer-local-binary-precedence.md
  - created: .pose/reports/2026-08-07-standard-validate-native.md
  - created: .pose/specs/pose-installer-local-binary-precedence/spec.md
  - created: pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-installer-local-binary-precedence.md
  - created: pose-mcp/internal/scaffold/dist/.pose/specs/pose-installer-local-binary-precedence/spec.md
  - modified: .pose/reports/history/standard-validate-native.jsonl
  - modified: .pose/results/delivery-validation.json
  - modified: .pose/state/history.jsonl
  - modified: .pose/state/project-state.md
  - modified: .pose/state/refresh-log.jsonl
  - modified: install.sh
  - modified: pose-mcp/internal/scaffold/dist/.pose/results/delivery-validation.json
  - modified: pose-mcp/internal/scaffold/dist/install.sh
  - modified: tests/install/run.sh

## Execution Metadata
- Generated at (UTC): 2026-08-07T04:18:46Z
- Context: closeout
- Validation profile: strict
- Sequence for task/spec: 1
- Stable comparison hash: 8245443321673fcd5d9d8c8cfcbc3929ffef8a7d80614a80a60ca4e6997b28f3

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
