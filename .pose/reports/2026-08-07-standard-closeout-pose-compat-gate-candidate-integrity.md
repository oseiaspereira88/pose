# POSE Report - 2026-08-07

## Report Type
- standard

## Task
- closeout-pose-compat-gate-candidate-integrity
- Task slug: closeout-pose-compat-gate-candidate-integrity
- Spec: pose-compat-gate-candidate-integrity

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
- ID: cs-6b153fe9345a
- Selector: range:fd66001edeb29c10bbdeed6457b90bcc96b391bf..848808e
- Base: fd66001edeb29c10bbdeed6457b90bcc96b391bf (fd66001edeb29c10bbdeed6457b90bcc96b391bf)
- Head: 848808e (848808eecd90a3325d38d5411440c19a3fe99b5c)
- Diff digest: sha256:bb8ab75686831bf5a0d8674790cde25ca2b5f7a61b1234caa3aefb84f17ccc8c
- Paths:
  - created: .pose/changelogs/unreleased/pose-compat-gate-candidate-integrity.md
  - created: .pose/specs/pose-compat-gate-candidate-integrity/spec.md
  - created: pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-compat-gate-candidate-integrity.md
  - created: pose-mcp/internal/scaffold/dist/.pose/specs/pose-compat-gate-candidate-integrity/spec.md
  - modified: .pose/state/history.jsonl
  - modified: .pose/state/project-state.md
  - modified: .pose/state/refresh-log.jsonl
  - modified: compatibility.json
  - modified: pose-mcp/internal/cli/install.go
  - modified: pose-mcp/internal/cli/managed_docs.go
  - modified: pose-mcp/internal/cli/managed_docs_test.go
  - modified: tests/release/compat.sh

## Execution Metadata
- Generated at (UTC): 2026-08-07T04:37:08Z
- Context: closeout
- Validation profile: strict
- Sequence for task/spec: 1
- Stable comparison hash: bd40d6795acdeb7c5059e86cb5f7636bd3cef5108e337466138e45e324da6870

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
