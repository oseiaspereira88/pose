# POSE Report - 2026-09-28

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
- pose/assessments/README.md
- .pose/assessments/consolidated.md
- .pose/assessments/docs-site.md
- .pose/assessments/integrations.md
- .pose/assessments/mcp-enforce.md
- .pose/assessments/pose-mcp.md
- .pose/assessments/technical-debt.md
- .pose/indexes/delivery-integrity.json
- .pose/indexes/releases.json
- .pose/indexes/spec-graph.json
- .pose/results/abm-dependency-review.json
- .pose/results/delivery-validation.json
- .pose/state/components/docs-site.json
- .pose/state/components/mcp-enforce.json
- .pose/state/components/pose-mcp.json
- .pose/state/integrations.json
- .pose/state/project-state.md
- .pose/state/technical-debt.json
- .pose/review-attestations/rva-16b93d5350d97a79.json
- .pose/review-attestations/rva-25156bc96a9fef0d.json
- .pose/review-attestations/rva-366e4d80266d2d4e.json
- .pose/review-attestations/rva-3b792f9063f2790e.json
- .pose/review-attestations/rva-42bf3f36e5a773c4.json
- .pose/review-attestations/rva-6e297308dd1a1430.json
- .pose/review-attestations/rva-77f1b757029da7fe.json
- .pose/review-attestations/rva-7e76171b0e715cc0.json
- .pose/review-attestations/rva-9e3a84d33b9f78ca.json
- .pose/review-attestations/rva-aa038d9f09d00196.json
- .pose/review-attestations/rva-ab74da9658285ef9.json
- .pose/review-attestations/rva-b6d1a87638f7fdb6.json
- .pose/review-attestations/rva-c22fd4f7248f86b4.json
- .pose/review-attestations/rva-f068a8f030c16e16.json
- .pose/review-bundles/rvb-168baa62179bbe37.json
- .pose/review-bundles/rvb-1d5e2644c0b2fc07.json
- .pose/review-bundles/rvb-2bb2c3fdb0f41c61.json
- .pose/review-bundles/rvb-3868c8c1308d4b81.json
- .pose/review-bundles/rvb-4ad497c39e09ce6b.json
- .pose/review-bundles/rvb-50b2f7263d510b56.json
- .pose/review-bundles/rvb-5eeac008e002a7cb.json
- .pose/review-bundles/rvb-7f410fe83bb31c97.json
- .pose/review-bundles/rvb-885e9e395ca73115.json
- .pose/review-bundles/rvb-aad14734052d72b4.json
- .pose/review-bundles/rvb-b8881b0d72e5996f.json
- .pose/review-bundles/rvb-df461992562d7193.json
- .pose/review-bundles/rvb-e47dcb0a379e17b1.json
- .pose/review-bundles/rvb-ed25ace6522d1d3c.json

## Validation Commands
- go build ./...
- go test ./...
- go vet ./...
- go build ./...
- go test ./...
- go vet ./...
- go test ./internal/pose -run GovernanceReplay|ABMGolden -count=1
- go test ./internal/cli -run GovernanceReplayCLIReachability -count=1
- go test ./internal/pose ./internal/cli ./internal/scaffold -run BundlePathWontFix|NegativeDecisionIsAudit|ReuseCannotLaunder|SignedRiskIsNotApproval|WontFixCannotApprove|DistributedReviewSoundness|ReviewAttestationEnvelope -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run GovernanceOutcomes|GovernanceStats|RemediationProjection -count=1
- go test ./internal/pose ./internal/cli -run RemediationLineage|RemediationProjection -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run ABMContractNodes|ABMNodeTransitions|ABMNodeDigest|Amend -count=1
- go test ./internal/pose ./internal/cli ./internal/mcpserver -run ABMAtomicStart -count=1
- go test ./internal/pose ./internal/cli -run ABMCausalityCloseout -count=1
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
- go test ./internal/pose -run TestFederatedSpecAcceptance|TestLegacyReviewBundlePayloadOmitsFederatedManifest -count=1
- go test ./internal/pose ./internal/cli -run TestFederatedAcceptanceNegative -count=1
- go test ./internal/pose ./internal/cli -run TestSpecTransfer -count=1
- go test ./internal/pose ./internal/cli -run TestSpecTransferNegative|TestSpecTransferBoundary|TestSpecTransferPreviewBlocks|TestSpecTransferResumeBlocks|TestSpecTransferResolverRejects|TestSpecAuthorityTransferPolicySchema -count=1
- go test ./internal/mcpserver -run TestSpecTransferStatus -count=1
- go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentSurface -count=1
- go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgent -count=1
- go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentNegative -count=1
- go test ./internal/cli -run TestMultiRepoAgentInstalled -count=1

## Results
- - [pass] mcp-enforce/go/build (0.6s)
- - [pass] mcp-enforce/go/test (0.1s)
- - [pass] mcp-enforce/go/vet (0.1s)
- - [pass] pose-mcp/go/build (1.3s)
- - [pass] pose-mcp/go/test (40.2s)
- - [pass] pose-mcp/go/vet (0.4s)
- - [pass] pose-mcp/go/abm-retrospective-replay-integration (0.4s)
- - [pass] pose-mcp/go/abm-replay-reachability (0.6s)
- - [pass] pose-mcp/go/review-soundness-residuals-integration (0.7s)
- - [pass] pose-mcp/go/governance-outcomes-v2-contract (0.7s)
- - [pass] pose-mcp/go/abm-remediation-lineage-integration (1.0s)
- - [pass] pose-mcp/go/abm-contract-nodes-integration (0.8s)
- - [pass] pose-mcp/go/abm-atomic-start-integration (0.8s)
- - [pass] pose-mcp/go/abm-causality-closeout-integration (0.7s)
- - [pass] pose-mcp/go/abm-progressive-review-integration (0.7s)
- - [pass] pose-mcp/go/artifact-claim-boundary-integration (0.9s)
- - [pass] pose-mcp/go/check-verdict-mode-integration (0.7s)
- - [pass] pose-mcp/go/delivery-graph-reuse-integration (0.6s)
- - [pass] pose-mcp/go/parse-memo-integration (0.2s)
- - [pass] pose-mcp/go/parallel-gate-integration (2.1s)
- - [pass] pose-mcp/go/check-workers-integration (0.7s)
- - [pass] pose-mcp/go/review-policy-adoption-integration (0.7s)
- - [pass] pose-mcp/go/delivery-integration (1.0s)
- - [pass] pose-mcp/go/delivery-reachability (0.8s)
- - [pass] pose-mcp/go/review-bundle-convergence (0.9s)
- - [pass] pose-mcp/go/qualified-artifact-resolution-integration (0.7s)
- - [pass] pose-mcp/go/federated-roadmap-acceptance-integration (7.5s)
- - [pass] pose-mcp/go/federated-spec-acceptance-integration (0.8s)
- - [pass] pose-mcp/go/federated-roadmap-negative-gates (0.8s)
- - [pass] pose-mcp/go/spec-authority-transfer-integration (1.5s)
- - [pass] pose-mcp/go/spec-authority-transfer-negative-gates (0.8s)
- - [pass] pose-mcp/go/spec-authority-transfer-mcp-status (0.8s)
- - [pass] pose-mcp/go/multi-repo-agent-context-reachability (0.9s)
- - [pass] pose-mcp/go/multi-repo-agent-context-integration (2.1s)
- - [pass] pose-mcp/go/multi-repo-agent-negative-gates (0.9s)
- - [pass] pose-mcp/go/multi-repo-agent-installed-journey (1.6s)
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-09-28T12:46:58Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 204
- Stable comparison hash: 5b47855e60f64e73728abd99582eb01357a94f0c289ad7fa9125d680a322e54f

## Historical Comparison
- Previous execution: 2026-09-28T05:10:22Z
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
