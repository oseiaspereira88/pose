# POSE Report - 2026-09-25

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
- .pose/assessments/technical-debt.md
- .pose/indexes/delivery-integrity.json
- .pose/reports/history/standard-validate-native.jsonl
- .pose/results/delivery-validation.json
- .pose/state/integrations.json
- .pose/state/technical-debt.json
- .pose/reports/2026-09-25-standard-validate-native.md

## Validation Commands
- go build ./...
- go test ./...
- go vet ./...
- go test ./internal/pose ./internal/cli -run RemediationLineage|RemediationProjection -count=1
- go test ./internal/pose ./internal/cli -run ABMProgressiveReview|ABMCriticality|ABMPolicyDowngrade|ABMProtectedBaseline|ABMStructural -count=1
- go test ./internal/pose -run ArtifactClaims -count=1
- go test ./internal/cli -run CheckVerdictMode -count=1
- go test ./internal/cli -run FocusSurfaceGraph|DeliverySpecBlockersFromShared -count=1
- go test ./internal/pose -run DeliveryGraphCache|DeliveryGraphCopy|DesignDeltaMemo|DesignDeltaCopy -count=1
- go test -race ./internal/cli -run FailOrWarnPerItem -count=1
- go test ./internal/cli ./internal/version ./internal/pose -run CheckWorkerCount|CompositionContract|PublishedRootManifests -count=1
- go test ./internal/cli -run InstallStampsItsOwnReviewAdoption|ShippedReviewPolicyCarriesNoAdoptionDate|UpdateKeepsRecordedAdoption -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run Delivery|Surface|RoadmapCheck|Contributor -count=1
- go test ./internal/cli ./internal/mcpserver -run Surface|DeliveryIntegrity|RoadmapCheck|Contributor -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run ReviewBundle|ReviewAttestation|ReviewPlanGroupsRepeatedWarnings|ReviewPlanActionableToolPhases|ToolCatalog -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run TestQualifiedArtifact -count=1
- go test ./internal/pose ./internal/cli -run TestFederatedRoadmap|TestFederatedAcceptance -count=1
- go test ./internal/pose ./internal/cli -run TestFederatedAcceptanceNegative -count=1
- go test ./internal/pose ./internal/cli -run TestSpecTransfer -count=1
- go test ./internal/pose ./internal/cli -run TestSpecTransferNegative|TestSpecTransferBoundary|TestSpecTransferPreviewBlocks|TestSpecTransferResumeBlocks|TestSpecTransferResolverRejects|TestSpecAuthorityTransferPolicySchema -count=1
- go test ./internal/mcpserver -run TestSpecTransferStatus -count=1
- go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentSurface -count=1
- go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgent -count=1
- go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentNegative -count=1
- go test ./internal/cli -run TestMultiRepoAgentInstalled -count=1

## Results
- - [pass] pose-mcp/go/build (0.9s)
- - [pass] pose-mcp/go/test (3.2s)
- - [pass] pose-mcp/go/vet (0.1s)
- - [pass] pose-mcp/go/abm-remediation-lineage-integration (1.0s)
- - [pass] pose-mcp/go/abm-progressive-review-integration (0.9s)
- - [pass] pose-mcp/go/artifact-claim-boundary-integration (1.0s)
- - [pass] pose-mcp/go/check-verdict-mode-integration (0.8s)
- - [pass] pose-mcp/go/delivery-graph-reuse-integration (0.6s)
- - [pass] pose-mcp/go/parse-memo-integration (0.1s)
- - [pass] pose-mcp/go/parallel-gate-integration (1.7s)
- - [pass] pose-mcp/go/check-workers-integration (0.6s)
- - [pass] pose-mcp/go/review-policy-adoption-integration (0.6s)
- - [pass] pose-mcp/go/delivery-integration (0.8s)
- - [pass] pose-mcp/go/delivery-reachability (0.6s)
- - [pass] pose-mcp/go/review-bundle-convergence (0.8s)
- - [pass] pose-mcp/go/qualified-artifact-resolution-integration (0.7s)
- - [pass] pose-mcp/go/federated-roadmap-acceptance-integration (0.8s)
- - [pass] pose-mcp/go/federated-roadmap-negative-gates (0.6s)
- - [pass] pose-mcp/go/spec-authority-transfer-integration (0.8s)
- - [pass] pose-mcp/go/spec-authority-transfer-negative-gates (0.5s)
- - [pass] pose-mcp/go/spec-authority-transfer-mcp-status (0.5s)
- - [pass] pose-mcp/go/multi-repo-agent-context-reachability (0.5s)
- - [pass] pose-mcp/go/multi-repo-agent-context-integration (1.4s)
- - [pass] pose-mcp/go/multi-repo-agent-negative-gates (0.6s)
- - [pass] pose-mcp/go/multi-repo-agent-installed-journey (1.4s)
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-09-25T01:02:31Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 167
- Stable comparison hash: 5b47855e60f64e73728abd99582eb01357a94f0c289ad7fa9125d680a322e54f

## Historical Comparison
- Previous execution: 2026-09-25T01:00:27Z
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
