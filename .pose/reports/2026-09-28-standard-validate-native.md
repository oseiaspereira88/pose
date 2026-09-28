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
- pose/indexes/delivery-integrity.json
- .pose/reports/2026-09-28-standard-validate-native.md
- .pose/reports/history/standard-validate-native.jsonl
- .pose/results/abm-dependency-review.json
- .pose/results/delivery-validation.json
- .pose/review-attestations/rva-006ead3e48f6f449.json
- .pose/review-attestations/rva-01c35f112a4832a8.json
- .pose/review-attestations/rva-06bccb6217cb0a62.json
- .pose/review-attestations/rva-06fa79edb0b31baf.json
- .pose/review-attestations/rva-0702b44d71bf4a89.json
- .pose/review-attestations/rva-0f8527d67944eef2.json
- .pose/review-attestations/rva-11051f73f3c2d48d.json
- .pose/review-attestations/rva-136fbe28df261c02.json
- .pose/review-attestations/rva-143f66cc70946edd.json
- .pose/review-attestations/rva-1864f753c6b360e2.json
- .pose/review-attestations/rva-1b6b7f8b7ce4be39.json
- .pose/review-attestations/rva-2195cd716267e450.json
- .pose/review-attestations/rva-2f603d93f2d91c12.json
- .pose/review-attestations/rva-30c017ba28eb66ed.json
- .pose/review-attestations/rva-35fafe29af0fd5e6.json
- .pose/review-attestations/rva-37cc04711f6f7664.json
- .pose/review-attestations/rva-3b920f66b9c14dac.json
- .pose/review-attestations/rva-41c88c052acaed9c.json
- .pose/review-attestations/rva-49a78548cdc30cfd.json
- .pose/review-attestations/rva-5e0bf432366ad62e.json
- .pose/review-attestations/rva-5f0a3667d8aa6084.json
- .pose/review-attestations/rva-5fa2673983b2818c.json
- .pose/review-attestations/rva-634c7bfe15c1cf1c.json
- .pose/review-attestations/rva-67c5904d37b828ab.json
- .pose/review-attestations/rva-7619820d5b4dac4d.json
- .pose/review-attestations/rva-8ee2e7f920567f88.json
- .pose/review-attestations/rva-9b11ca399d046589.json
- .pose/review-attestations/rva-ab992bfd11da8d57.json
- .pose/review-attestations/rva-ac1f2e05a1096235.json
- .pose/review-attestations/rva-bd96703a3bda08c0.json
- .pose/review-attestations/rva-c27ae4d936f2130e.json
- .pose/review-attestations/rva-d2637ecdce857069.json
- .pose/review-attestations/rva-db94cc2074eee039.json
- .pose/review-attestations/rva-eaffe9b7512c9a5d.json
- .pose/review-bundles/rvb-02cddbf58f26b615.json
- .pose/review-bundles/rvb-0302fb63ce24e8a4.json
- .pose/review-bundles/rvb-20c1a49678e17b52.json
- .pose/review-bundles/rvb-2309cf6caae0767f.json
- .pose/review-bundles/rvb-38867009a7854224.json
- .pose/review-bundles/rvb-4b6590453127e43a.json
- .pose/review-bundles/rvb-56187f69b3c6aee9.json
- .pose/review-bundles/rvb-5800651946126a5c.json
- .pose/review-bundles/rvb-596c17da4e14d24a.json
- .pose/review-bundles/rvb-5ab332d48c618791.json
- .pose/review-bundles/rvb-626f1b8b0c803bc7.json
- .pose/review-bundles/rvb-6597875ebc9e6d70.json
- .pose/review-bundles/rvb-66002e53e4de506a.json
- .pose/review-bundles/rvb-6b1f6e4bb7afe00b.json
- .pose/review-bundles/rvb-6d3e8638bc7c7e63.json
- .pose/review-bundles/rvb-74817a6a3ceabdfc.json
- .pose/review-bundles/rvb-8c32284b62a3db02.json
- .pose/review-bundles/rvb-91ab16b2b024fdb0.json
- .pose/review-bundles/rvb-95ffb1f62cba4191.json
- .pose/review-bundles/rvb-afa9aeb96b0a784a.json
- .pose/review-bundles/rvb-b02464a2fbe95500.json
- .pose/review-bundles/rvb-b57dd5845e345251.json
- .pose/review-bundles/rvb-b67f9fd5365520d2.json
- .pose/review-bundles/rvb-c250d5874bf9e243.json
- .pose/review-bundles/rvb-c75a7041102f7332.json
- .pose/review-bundles/rvb-ca974a9f5386caf2.json
- .pose/review-bundles/rvb-cce08385747695c3.json
- .pose/review-bundles/rvb-d932be028fb57a41.json
- .pose/review-bundles/rvb-d9fee4351d5af4c2.json
- .pose/review-bundles/rvb-df1882a3c4a6e2cc.json
- .pose/review-bundles/rvb-ebea7706a6335acd.json
- .pose/review-bundles/rvb-eed1627862a4b547.json
- .pose/review-bundles/rvb-f4ec72b26e63e3a2.json
- .pose/review-bundles/rvb-f6e275d557025e31.json

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
- - [pass] mcp-enforce/go/build (0.3s)
- - [pass] mcp-enforce/go/test (0.1s)
- - [pass] mcp-enforce/go/vet (0.1s)
- - [pass] pose-mcp/go/build (0.8s)
- - [pass] pose-mcp/go/test (7.1s)
- - [pass] pose-mcp/go/vet (0.2s)
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
- - [pass] pose-mcp/go/check-verdict-mode-integration (0.8s)
- - [pass] pose-mcp/go/delivery-graph-reuse-integration (0.6s)
- - [pass] pose-mcp/go/parse-memo-integration (0.1s)
- - [pass] pose-mcp/go/parallel-gate-integration (2.2s)
- - [pass] pose-mcp/go/check-workers-integration (0.7s)
- - [pass] pose-mcp/go/review-policy-adoption-integration (0.7s)
- - [pass] pose-mcp/go/delivery-integration (1.1s)
- - [pass] pose-mcp/go/delivery-reachability (1.0s)
- - [pass] pose-mcp/go/review-bundle-convergence (0.9s)
- - [pass] pose-mcp/go/qualified-artifact-resolution-integration (0.7s)
- - [pass] pose-mcp/go/federated-roadmap-acceptance-integration (7.3s)
- - [pass] pose-mcp/go/federated-spec-acceptance-integration (0.8s)
- - [pass] pose-mcp/go/federated-roadmap-negative-gates (0.7s)
- - [pass] pose-mcp/go/spec-authority-transfer-integration (1.4s)
- - [pass] pose-mcp/go/spec-authority-transfer-negative-gates (0.7s)
- - [pass] pose-mcp/go/spec-authority-transfer-mcp-status (0.8s)
- - [pass] pose-mcp/go/multi-repo-agent-context-reachability (0.7s)
- - [pass] pose-mcp/go/multi-repo-agent-context-integration (1.8s)
- - [pass] pose-mcp/go/multi-repo-agent-negative-gates (0.8s)
- - [pass] pose-mcp/go/multi-repo-agent-installed-journey (1.6s)
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-09-28T13:33:44Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 209
- Stable comparison hash: 5b47855e60f64e73728abd99582eb01357a94f0c289ad7fa9125d680a322e54f

## Historical Comparison
- Previous execution: 2026-09-28T13:32:13Z
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
