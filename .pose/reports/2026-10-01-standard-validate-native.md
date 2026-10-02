# POSE Report - 2026-10-01

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
- .pose/assessments/technical-debt.md
- .pose/reports/2026-10-01-standard-validate-native.md
- .pose/reports/history/standard-validate-native.jsonl
- .pose/state/technical-debt.json
- .pose/investigations/

## Validation Commands
- go build ./...
- go test ./...
- go vet ./...
- go test ./internal/pose -run ReviewReleaseArchive -count=1
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
- go test ./internal/cli -run AnyDirectoryName|DeclaredProjectID|InvalidStampedProjectID|InvalidExplicitProjectID|NeverResolved -count=1
- go test ./internal/cli -run CompletionOneDayBeforeUTCCreation -count=1
- go test ./internal/testgit -count=1
- go test ./internal/version -run ReleaseWorkflowWaitsForCI|ReleaseNeedsCIFindings -count=1
- go test ./internal/pose -run CompletedReviewRetention|FederatedRoadmapMilestoneBundleSealsOwnManifest|ReviewCheckNamesANegativeVerdict -count=1
- go test ./internal/pose -run ReviewSubjectClassifiesTheCapabilityAssessment -count=1
- go test ./internal/cli -run PublicClaimsReadsAMajorReleaseLine|EveryDocsPageThatNamesAReleaseLineIsADeclaredSurface -count=1
- go test ./internal/version ./internal/cli -run CIGovernanceJobRunsLintSpecAll|ReleasePrepareFindsADatePrefixedSpec -count=1
- go test ./internal/cli -run LintSpecJSON|LintSpecQuiet|LintSpecHumanOutput|IndexJSON|EveryGateOnTheMachineChannel|JSONOutWrites|ValidateJSONPathIsADeprecatedAlias|ReportChangedFilesKeepsTheFirstPath|QuietCheckPrintsTheVerdictAlone -count=1
- go test ./internal/usage ./internal/cli ./internal/mcpserver -run Verdict|UsageAdjudicate|UsageMCPReportsHumanVerdicts|ValidateUsageUsesStructuredCheckFindings -count=1
- go test ./internal/cli ./internal/mcpserver -run UsageAdjudicate|UsageMCPReportsHumanVerdicts -count=1
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
- - [pass] pose-mcp/go/build (1.4s)
- - [pass] pose-mcp/go/test (40.7s)
- - [pass] pose-mcp/go/vet (0.2s)
- - [pass] pose-mcp/go/release-review-archive-integration (0.6s)
- - [pass] pose-mcp/go/abm-retrospective-replay-integration (0.4s)
- - [pass] pose-mcp/go/abm-replay-reachability (0.7s)
- - [pass] pose-mcp/go/review-soundness-residuals-integration (0.8s)
- - [pass] pose-mcp/go/governance-outcomes-v2-contract (1.3s)
- - [pass] pose-mcp/go/abm-remediation-lineage-integration (1.3s)
- - [pass] pose-mcp/go/abm-contract-nodes-integration (1.3s)
- - [pass] pose-mcp/go/abm-atomic-start-integration (1.2s)
- - [pass] pose-mcp/go/abm-causality-closeout-integration (0.9s)
- - [pass] pose-mcp/go/abm-progressive-review-integration (1.0s)
- - [pass] pose-mcp/go/artifact-claim-boundary-integration (1.4s)
- - [pass] pose-mcp/go/check-verdict-mode-integration (1.2s)
- - [pass] pose-mcp/go/project-id-derivation-integration (1.1s)
- - [pass] pose-mcp/go/calendar-date-skew-integration (0.9s)
- - [pass] pose-mcp/go/test-git-isolation-integration (0.1s)
- - [pass] pose-mcp/go/release-needs-ci-contract (0.6s)
- - [pass] pose-mcp/go/completed-review-retention-integration (2.7s)
- - [pass] pose-mcp/go/review-subject-capabilities-integration (0.2s)
- - [pass] pose-mcp/go/public-claims-release-line-integration (0.9s)
- - [pass] pose-mcp/go/lint-spec-all-gate-contract (1.2s)
- - [pass] pose-mcp/go/machine-channel-integration (1.7s)
- - [pass] pose-mcp/go/usage-adjudication-integration (1.1s)
- - [pass] pose-mcp/go/usage-adjudication-reachability (0.9s)
- - [pass] pose-mcp/go/delivery-graph-reuse-integration (0.8s)
- - [pass] pose-mcp/go/parse-memo-integration (0.2s)
- - [pass] pose-mcp/go/parallel-gate-integration (6.8s)
- - [pass] pose-mcp/go/check-workers-integration (1.3s)
- - [pass] pose-mcp/go/review-policy-adoption-integration (0.8s)
- - [pass] pose-mcp/go/delivery-integration (1.3s)
- - [pass] pose-mcp/go/delivery-reachability (1.1s)
- - [pass] pose-mcp/go/review-bundle-convergence (1.2s)
- - [pass] pose-mcp/go/qualified-artifact-resolution-integration (1.1s)
- - [pass] pose-mcp/go/federated-roadmap-acceptance-integration (8.9s)
- - [pass] pose-mcp/go/federated-spec-acceptance-integration (0.9s)
- - [pass] pose-mcp/go/federated-roadmap-negative-gates (1.1s)
- - [pass] pose-mcp/go/spec-authority-transfer-integration (1.7s)
- - [pass] pose-mcp/go/spec-authority-transfer-negative-gates (1.2s)
- - [pass] pose-mcp/go/spec-authority-transfer-mcp-status (1.0s)
- - [pass] pose-mcp/go/multi-repo-agent-context-reachability (1.6s)
- - [pass] pose-mcp/go/multi-repo-agent-context-integration (2.7s)
- - [pass] pose-mcp/go/multi-repo-agent-negative-gates (1.2s)
- - [pass] pose-mcp/go/multi-repo-agent-installed-journey (2.2s)
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-10-01T21:24:46Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 217
- Stable comparison hash: 5b47855e60f64e73728abd99582eb01357a94f0c289ad7fa9125d680a322e54f

## Historical Comparison
- Previous execution: 2026-10-01T19:04:37Z
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
