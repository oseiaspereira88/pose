# POSE Report - 2026-10-21

## Report Type
- standard

## Task
- b07-snapshot-integrity
- Task slug: b07-snapshot-integrity

## Outcome
- Outcome: pass (source: derived)

## Rules Applied
- _Not provided_

## Files Changed
- .pose/assessments/README.md
- .pose/assessments/consolidated.md
- .pose/assessments/pose-mcp.md
- .pose/indexes/delivery-integrity.json
- .pose/indexes/generation.json
- .pose/indexes/roadmaps.json
- .pose/indexes/spec-graph.json
- .pose/reports/2026-10-20-b07-implementation.md
- .pose/reports/history/standard-b07-snapshot-integrity.jsonl
- .pose/results/delivery-validation.json
- .pose/specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md
- .pose/specs/2026-10-20-pose-retire-unused-first-party-action-exception.md
- .pose/state/components/pose-mcp.json
- .pose/state/project-state.md
- .pose/reports/2026-10-21-standard-b07-snapshot-integrity.md
- .pose/reports/pose-validate.latest.log
- .pose/review-attestations/rva-476a47d625b27e1a.json
- .pose/review-attestations/rva-d25bd22736b23e42.json
- .pose/review-bundles/rvb-0ae879ef12b5b804.json
- .pose/review-bundles/rvb-5419e7c7ab90aebe.json

## Validation Commands
- build go build ./...
- test go test ./...
- vet go vet ./...
- build go build ./...
- test go test ./...
- vet go vet ./...
- release-review-archive-integration go test ./internal/pose -run ReviewReleaseArchive -count=1
- abm-retrospective-replay-integration go test ./internal/pose -run GovernanceReplay|ABMGolden -count=1
- abm-replay-reachability go test ./internal/cli -run GovernanceReplayCLIReachability -count=1
- review-soundness-residuals-integration go test ./internal/pose ./internal/cli ./internal/scaffold -run BundlePathWontFix|NegativeDecisionIsAudit|ReuseCannotLaunder|SignedRiskIsNotApproval|WontFixCannotApprove|DistributedReviewSoundness|ReviewAttestationEnvelope -count=1
- governance-outcomes-v2-contract go test ./internal/pose ./internal/cli ./internal/mcpserver -run GovernanceOutcomes|GovernanceStats|RemediationProjection -count=1
- abm-remediation-lineage-integration go test ./internal/pose ./internal/cli -run RemediationLineage|RemediationProjection -count=1
- abm-contract-nodes-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run ABMContractNodes|ABMNodeTransitions|ABMNodeDigest|Amend -count=1
- abm-atomic-start-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run ABMAtomicStart -count=1
- abm-causality-closeout-integration go test ./internal/pose ./internal/cli -run ABMCausalityCloseout -count=1
- abm-progressive-review-integration go test ./internal/pose ./internal/cli -run ABMProgressiveReview|ABMCriticality|ABMPolicyDowngrade|ABMProtectedBaseline|ABMStructural -count=1
- artifact-claim-boundary-integration go test ./internal/pose -run ArtifactClaims -count=1
- check-verdict-mode-integration go test ./internal/cli -run CheckVerdictMode -count=1
- project-id-derivation-integration go test ./internal/cli -run AnyDirectoryName|DeclaredProjectID|InvalidStampedProjectID|InvalidExplicitProjectID|NeverResolved -count=1
- calendar-date-skew-integration go test ./internal/cli -run CompletionOneDayBeforeUTCCreation -count=1
- test-git-isolation-integration go test ./internal/testgit -count=1
- release-needs-ci-contract go test ./internal/version -run ReleaseWorkflowWaitsForCI|ReleaseNeedsCIFindings -count=1
- attention-snapshot-integration go test ./internal/pose -run AttentionSnapshot|TestObligation|AttentionSource -count=1
- recurrence-flapping-integration go test ./internal/cli -run TestRecurrence -count=1
- attention-sources-integration go test ./internal/pose ./internal/cli -run AttentionSource|FreshInstallHealthAttention|TestAPhaseWithAnUnreadProducerIsUnknownNotClear|TestAdversarialCorpus -count=1
- check-git-spawns-integration go test ./internal/pose -run CheckGitSpawns -count=1
- range-ownership-integration go test ./internal/pose -run SubjectRange -count=1
- locale-obligation-flags-integration go test ./internal/scaffold -run LocaleCoverage|LocaleObligationFlags -count=1
- trace-test-refs-integration go test ./internal/pose ./internal/cli -run TraceTestRefs -count=1
- red-signal-contract go test ./internal/version -run FailureAlert -count=1
- completed-review-retention-integration go test ./internal/pose -run CompletedReviewRetention|FederatedRoadmapMilestoneBundleSealsOwnManifest|ReviewCheckNamesANegativeVerdict -count=1
- review-subject-capabilities-integration go test ./internal/pose -run ReviewSubjectClassifiesTheCapabilityAssessment -count=1
- public-claims-release-line-integration go test ./internal/cli -run PublicClaimsReadsAMajorReleaseLine|EveryDocsPageThatNamesAReleaseLineIsADeclaredSurface -count=1
- lint-spec-all-gate-contract go test ./internal/version ./internal/cli -run CIGovernanceJobRunsLintSpecAll|ReleasePrepareFindsADatePrefixedSpec -count=1
- machine-channel-integration go test ./internal/cli -run LintSpecJSON|LintSpecQuiet|LintSpecHumanOutput|IndexJSON|EveryGateOnTheMachineChannel|JSONOutWrites|ValidateJSONPathIsADeprecatedAlias|ReportChangedFilesKeepsTheFirstPath|QuietCheckPrintsTheVerdictAlone -count=1
- usage-adjudication-integration go test ./internal/usage ./internal/cli ./internal/mcpserver -run Verdict|UsageAdjudicate|UsageMCPReportsHumanVerdicts|ValidateUsageUsesStructuredCheckFindings -count=1
- usage-adjudication-reachability go test ./internal/cli ./internal/mcpserver -run UsageAdjudicate|UsageMCPReportsHumanVerdicts -count=1
- delivery-graph-reuse-integration go test ./internal/cli -run FocusSurfaceGraph|DeliverySpecBlockersFromShared -count=1
- parse-memo-integration go test ./internal/pose -run DeliveryGraphCache|DeliveryGraphCopy|DesignDeltaMemo|DesignDeltaCopy -count=1
- parallel-gate-integration go test -race ./internal/cli -run FailOrWarnPerItem -count=1
- check-workers-integration go test ./internal/cli ./internal/version ./internal/pose -run CheckWorkerCount|CompositionContract|PublishedRootManifests -count=1
- review-policy-adoption-integration go test ./internal/cli -run InstallStampsItsOwnReviewAdoption|ShippedReviewPolicyCarriesNoAdoptionDate|UpdateKeepsRecordedAdoption -count=1
- delivery-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run Delivery|Surface|RoadmapCheck|Contributor -count=1
- delivery-reachability go test ./internal/cli ./internal/mcpserver -run Surface|DeliveryIntegrity|RoadmapCheck|Contributor -count=1
- review-bundle-convergence go test ./internal/pose ./internal/cli ./internal/mcpserver -run ReviewBundle|ReviewAttestation|ReviewPlanGroupsRepeatedWarnings|ReviewPlanActionableToolPhases|ToolCatalog -count=1
- qualified-artifact-resolution-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run TestQualifiedArtifact -count=1
- federated-roadmap-acceptance-integration go test ./internal/pose ./internal/cli -run TestFederatedRoadmap|TestFederatedAcceptance -count=1
- federated-spec-acceptance-integration go test ./internal/pose -run TestFederatedSpecAcceptance|TestLegacyReviewBundlePayloadOmitsFederatedManifest -count=1
- federated-roadmap-negative-gates go test ./internal/pose ./internal/cli -run TestFederatedAcceptanceNegative -count=1
- spec-authority-transfer-integration go test ./internal/pose ./internal/cli -run TestSpecTransfer -count=1
- spec-authority-transfer-negative-gates go test ./internal/pose ./internal/cli -run TestSpecTransferNegative|TestSpecTransferBoundary|TestSpecTransferPreviewBlocks|TestSpecTransferResumeBlocks|TestSpecTransferResolverRejects|TestSpecAuthorityTransferPolicySchema -count=1
- spec-authority-transfer-mcp-status go test ./internal/mcpserver -run TestSpecTransferStatus -count=1
- multi-repo-agent-context-reachability go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentSurface -count=1
- multi-repo-agent-context-integration go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgent -count=1
- multi-repo-agent-negative-gates go test ./internal/cli ./internal/mcpserver -run TestMultiRepoAgentNegative -count=1
- multi-repo-agent-installed-journey go test ./internal/cli -run TestMultiRepoAgentInstalled -count=1
- dependabot-runtime-repair-integration go test ./internal/version -run TestDependabotRuntimeRepairTrustBoundary -count=1
- ci-release-obligations-integration go test ./internal/version -run TestLocalVerify|TestReleaseObligationsAndNegativeControls -count=1
- cli-profile-parity-integration go test ./internal/cli/cliout -run TestLifecycleRendererProfileGoldens|TestColouredTablesKeepPlainAlignment|TestUTF8ProseWrapsByCharacters|TestUnknownKindsAreFullyLocalised -count=1
- extension-target-integration go test ./internal/cli -run TestExtensionInstallTargetFlag|TestExtensionInstallRejectsInvalidTargetBeforeLookup|TestExtensionInstallDryRunShowsDefaultTargetWithoutWriting -count=1
- release-version-source-integration go test ./internal/pose ./internal/cli -run ReleaseVersionSource|ReleasePolicyOmitsAnUnsetVersionSource|LoadReleasePolicyValidatesTheVersionSource|TestReleasePrepare|TestReleasePlan|TestReleaseCheck -count=1
- delivery-integrity-index-compaction-integration go test ./internal/pose ./internal/cli -run CompactIndex|DeliveryIntegrityIndexReads|StoreReadsTheIndexItWroteInSchemaTwo|TestIndexWrites -count=1
- release-version-source-reachability go test ./internal/cli -run TestReleaseVersionSource -count=1
- clean-activation-recording-integration go test ./internal/version -run TestActivationRecordingContract -count=1
- legacy-contract-cutoffs-integration go test ./internal/pose ./internal/cli -run TestContractAdopt|TestLegacyKeyCutoffs|TestLoadReviewPolicyRejectsAnUnparsableRegisteredAdoptionDate|TestDoctorReports(Legacy|AShadowed) -count=1
- legacy-contract-cutoffs-reachability go test ./internal/cli -run TestDoctorReports(Legacy|AShadowed) -count=1
- public-claims-provenance-integration go test ./internal/cli -run TestPublicClaims -count=1
- public-claims-provenance-reachability go test ./internal/cli -run TestPublicClaimsReportsAPreparedCandidate|TestPublicClaimsWithoutRetainedPublication|TestPublicClaimsRejectsAPublicationWithLifecycleGaps -count=1
- review-assurance-disclosure-integration go test ./internal/pose ./internal/cli -run Assurance|TestDeclared|TestVerifiedAssurance|TestLegacyAttemptUnder|TestReviewSurfacesDisclose|TestNoSurfaceClaimsAnIndependentReview|TestABMReviewAuthority -count=1
- review-assurance-disclosure-reachability go test ./internal/cli -run TestReviewSurfacesDiscloseDeclaredAssuranceOfAHumanLabel -count=1
- review-attribution-roles-integration go test ./internal/pose ./internal/cli -run Attribution|630Cycle|SerializesAsBefore|OnlyAVerifiedHuman|ReviewConfirmation|TestReviewerFlagNeverFills|TestReviewAttestRecordsExplicitAttribution -count=1
- review-attribution-roles-reachability go test ./internal/cli -run TestReviewerFlagNeverFills|TestReviewAttestRecordsExplicitAttribution -count=1
- blocked-semantics-integration go test ./internal/pose ./internal/cli -run Blocked|TestAdoptionMetricsKeepsV1|TestLintWarnsOnBlocked|TestNoCommandRewrites|TestStateListsBlocked|TestSpecReadiness|Transfer -count=1
- blocked-semantics-reachability go test ./internal/cli -run TestAdoptionMetricsKeepsV1|TestLintWarnsOnBlocked|TestNoCommandRewrites -count=1
- flat-spec-amendments-integration go test ./internal/pose ./internal/cli -run Amend|TestSpecFormat|AmendmentsPath -count=1
- flat-spec-amendments-reachability go test ./internal/cli -run TestAmendWorksForFlat|TestSpecFormatMigrationCarries -count=1
- obligation-contract-integration go test ./internal/pose -run Obligation|Collides|UnknownIsNever|LegacyOpaque|SchemaDrift|EveryCategoryValidates -count=1
- effective-governance-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run Governance|Unadopted|ProjectionReads|ScopedProjection|ProjectionWrites|ProjectState -count=1
- effective-governance-reachability go test ./internal/cli ./internal/mcpserver -run TestStateGovernanceIsReachableAndReadOnly|TestToolsCall_ProjectState_CarriesEffectiveGovernance -count=1
- adversarial-corpus go test ./internal/cli -run TestAdversarialCorpus -count=1
- typed-diagnostics-integration go test ./internal/pose ./internal/cli -run Diagnostic|WaitingRefsCarry|JudgmentPendencyNames|CloseoutDiagnostics|TestCloseoutCheckJSONCarriesTypedDiagnostics|TestSpecReadiness|TestABMReviewSoundness -count=1
- typed-diagnostics-reachability go test ./internal/cli -run TestCloseoutCheckJSONCarriesTypedDiagnostics -count=1
- obligation-projection-integration go test ./internal/pose ./internal/cli -run Projection|Resolving|FailingProducer|Unintegrated|Correlation|SnapshotChanges|QueryFilters|TestAdversarialCorpus|Followup -count=1
- state-attention-reachability go test ./internal/cli ./internal/mcpserver -run TestStateAttentionShowsCoverageFirstAndSeparatesKinds|TestStateAttentionAndMCPShareIDs|TestToolsCall_Obligations_ReturnsReportWithCoverage -count=1
- action-requests-reachability go test ./internal/cli -run TestActionOpenPreviewsThenRecordsAndShowsTheBoundContent -count=1
- action-resolution-integration go test ./internal/pose ./internal/cli -run Actor|Declining|DeclaredHuman|CancelAndWaive|ReplayIs|StaleDigest|AnotherProject|VerifiedClaim|TestActionResolve|TestAdversarialCorpus -count=1
- action-resolution-reachability go test ./internal/cli -run TestActionResolveRefusesTheWrongActorAndRecordsTheRightOne -count=1
- phase-readiness-integration go test ./internal/pose ./internal/mcpserver ./internal/cli -run CloseoutOnlyRequest|LocalizedRestriction|UnreadProducer|LegacyReadyKeeps|PhasesBesideLegacy|TestAdversarialCorpus|TestSpecReadiness -count=1
- phase-readiness-reachability go test ./internal/mcpserver -run TestToolsCall_SpecReadiness_PhasesBesideLegacyReady -count=1
- governed-effects-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run WithoutAdoption|AdoptedEffects|DeclinedRequestKeeps|StartPreviewTaken|CloseRefusesWhile|CarriesTheGovernedEffect|Governance -count=1
- governed-effects-reachability go test ./internal/cli ./internal/mcpserver -run TestCloseRefusesWhileAnAdoptedRequestRestrictsCloseout|TestToolsCall_CloseoutState_CarriesTheGovernedEffect -count=1
- transfer-actions-integration go test ./internal/pose -run TransferInvalidates|TransferRefusesARequest|SpecTransfer -count=1
- closeout-plan-integration go test ./internal/cli -run ClosePlan|InterruptedCloseout|StopsAtJudgment|StalePlanDigest|TestAdversarialCorpus -count=1
- closeout-plan-reachability go test ./internal/cli -run TestAnInterruptedCloseoutResumesWithoutDuplication|TestClosePlanPreviewsWithoutWriting -count=1
- material-reuse-integration go test ./internal/pose ./internal/cli -run BookkeepingChanges|EquivalenceAgrees|Reuse|reuse -count=1
- assessment-freshness-integration go test ./internal/pose ./internal/cli -run AssessmentFreshness|StaleAssessment|TestNoWorkflowAlwaysRunsDiscovery|Discover -count=1
- followup-candidates-integration go test ./internal/pose ./internal/cli -run CandidatesAreRaised|FollowupCandidatesAreReachable|Followup -count=1
- followup-candidates-reachability go test ./internal/cli -run TestFollowupCandidatesAreReachableAndWriteNothing -count=1
- action-presentation-integration go test ./internal/pose ./internal/mcpserver -run RelatedRequestsAreGrouped|TestToolsList|TestCatalog -count=1
- spec-surface-integration go test ./internal/cli ./internal/pose -run MinimalSurface|LintAppliesTheSurface|SpecFactsAreDerived|TestNewSpec -count=1
- spec-surface-reachability go test ./internal/cli -run TestMinimalSurfaceKeepsGovernanceAndDropsRitual|TestLintAppliesTheSurfaceButNotToLifecycleGates -count=1
- premise-validity-integration go test ./internal/pose -run ContractChangeAsks|TrivialAssumptions|ValidityFieldsAreParsed|PortugueseValidity|TestABMDesignBasis -count=1
- falsifier-integration go test ./internal/pose -run FailedFalsifier|PassingOrAbsent|FalsifierFieldsAreOptIn|TestABMDesignBasis -count=1
- migrate-dryrun-integration go test ./internal/cli -run MigrateDryRun|TestHelp -count=1
- action-identity-integration go test ./internal/cli -run TestActionOpenDisclosesADirectoryDerivedIdentity|TestActionOpenPreviews -count=1
- adversarial-corpus-integration go test ./internal/cli -run TestAdversarialCorpus|TestDelegatedCorpusCasesStillExist -count=1
- governance-waits-integration go test ./internal/pose ./internal/cli -run WaitsSeparate|ReworkClassifies|MissingTimestamp|BookkeepingChanges|EquivalenceAgrees|TestGovernanceStats|WaitsAloneDoesNotComputeOutcomes -count=1
- action-requests-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run TestActionRequest|TestActionOpen|TestOpenActionRequest|TestSupersededRequest|TestToolsCall_ActionOpen|TestCatalog|TestToolsList -count=1
- governance-waits-reachability go test ./internal/cli -run TestWaitsAloneDoesNotComputeOutcomes -count=1
- migrate-dryrun-reachability go test ./internal/cli -run TestMigrateDryRunInventoriesLegacyWithoutWriting -count=1
- state-attention-integration go test ./internal/cli ./internal/mcpserver ./internal/pose -run Attention|TestToolsCall_Obligations|TestToolsList|TestCatalog -count=1
- atomic-start-cutoff-integration go test ./internal/pose -run AtomicStartCutoff|TestStart|PreviewStart|ApplyStart -count=1
- validation-check-materiality-integration go test ./internal/pose -run ValidationCheckMateriality -count=1
- causality-cutoff-integration go test ./internal/pose -run CausalityCutoff|ABMCausalityCloseout -count=1
- attest-preflight-integration go test ./internal/cli -run AttestPreflight|ReviewRecordDelegatesToBundleAttestation|ReviewAttestationWontFixCannotApproveThroughCLI -count=1
- governed-capabilities-integration go test ./internal/pose ./internal/cli -run GovernedCapabilit|Adopt|DoctorWarnsWhenAgency|ABMProgressiveReview -count=1
- authority-project-integration go test ./internal/pose -run AuthorityProject|ABMReviewAuthority|VerifiedClaim -count=1
- qualified-followup-integration go test ./internal/cli -run QualifiedFollowup -count=1
- recurrence-cluster-integration go test ./internal/cli -run Recurrence -count=1
- project-identity-integration go test ./internal/pose ./internal/cli -run ProjectFile -count=1
- capability-catalog-integration go test ./internal/pose ./internal/cli -run CapabilityCatalog|AdoptCatalog|GovernedCapabilit -count=1
- init-is-install-integration go test ./internal/cli -run InitIsInstall|InitOnAnInstalled|InitNative -count=1
- fresh-install-health-integration go test ./internal/pose ./internal/cli -run FreshInstallHealth|ShippedStack|APhaseWithAnUnreadProducer|DoctorWarnsWhenAgencyReadinessHasNoPrincipal|DoctorFindingsHaveEvidence -count=1
- signed-action-answers-integration go test ./internal/pose ./internal/cli -run SSHSig|PrincipalKey|SignedAnswer|Identity|DoctorReportsHumanRoles -count=1
- mcp-review-attest-integration go test ./internal/mcpserver ./internal/pose -run ReviewAttest|ReviewPrepare|SignedPreview|EnvelopeOverOther|AgentConfirmation|PreviewBindsOnly -count=1
- mcp-action-resolve-integration go test ./internal/mcpserver -run ActionResolve|Catalog|ToolsList -count=1
- setup-command-integration go test ./internal/cli -run Setup|CompatibilityUpgradeFromLegacyInstance -count=1
- onboarding-spec-integration go test ./internal/cli -run Onboarding -count=1
- configuration-review-integration go test ./internal/cli -run ConfigurationReview|SetupAnswersAndAppliesAReviewRequest -count=1
- quickstart-real-lifecycle-integration go test ./internal/cli -run CloseTraceGate|QuickstartScript|StackExtension -count=1
- install-and-upgrade-journeys-integration go test ./internal/cli -run Journeys -count=1
- historical-trust-reuse-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run ReviewTrustTime|ReviewReuseProvenance|ReviewTrustReuse|ActionRead|ActionClaim|ReviewAssurance|ReviewConsumersAgree|ABMReviewAuthority|NativeAuthorityClaim|NativeAndExternalIssuers|SignedPreview|AgentConfirmation -count=1
- review-approval-domain-preflight-integration go test ./internal/pose ./internal/cli -run ReviewDomainPreflight|ABMReviewSoundness|ABMGolden|LegacyLedger|CompletedReviewRetention|AuthorityProject|ReviewAttestationEnvelopeTrustPolicy -count=1
- cli-mcp-operation-results-integration go test ./internal/pose ./internal/cli ./internal/mcpserver ./internal/usage -run Inconclusive|StillAVerdict|OperationResultStarts|CloseApplyJSON|CloseAndStartRefuse|SpecsRefuses|ContinuousCloseoutStatusJSON|UsageCountsAWaiting|UsageCountsWaitingAndSkips|ListSpecsRefuses|EvidenceThatValidate|ARecentResult|ACheckThatPrints|ResultsThisRun|AStartThatFails|ARetriedStart|AResumeThatFails|AStaleStart|AStartReceipt|AStartWhoseTransition|AStartBaselineWriteCut|AnUnreadableStartRecord|JSONRoutesReport|SpecFactsRefuses|ClosePlan|InterruptedCloseout|StalePlanDigest|ABMAtomicStartCLI -count=1
- journal-checkpoint-and-lock-recovery-integration go test -race ./internal/pose ./internal/cli ./internal/mcpserver -run AJournal|AnInterruptedTail|ACheckpoint|AStartLockIsJudged|ALockWithoutA|OnlyAProvenDeadLock|ConcurrentRecoveries|AReleaseRemovesOnly|ARecoveredLock|TheStartCommandReports|ADeadStartLock|MCPLocates|MCPStartStatusShows|ABMAtomicStartConcurrent -count=1
- git-and-dispatch-snapshot-integrity-integration go test ./internal/pose ./internal/cli -run GitStateThatCannotBeRead|AssumeUnchangedAndSkipWorktree|WorktreeDivergenceReads|DerivedRegenerationIsAList|DispatchStopsGoverning|UnreadableSnapshotIsNotClean|UnreadableStatusIsNotACleanPath|ProjectionIsNotCoherentWhenStatus|DispatchJudgesTheContent|PolicyChangedWhileItRanIsStale|UnrelatedCommitDuringTheRun|ComparedWithTheSealedCommitBeforeTheRun -count=1
- writer-coordination-and-atomic-publication-integration go test -race ./internal/pose ./internal/cli -run ConcurrentPublishers|PublicationsKeep|AFailedPublication|ATransferWaits|StartRefusesWhile|TheActionPolicyIsWritten|ACloseNever|ACloseWaits|ACloseIsBound|ARefreshThatPublished|ARefreshNever|AHistoryEntry|SpecTransferConcurrentApply|AuthoritativePublicationsSync|AnUnsyncedPublication|AnImmutablePublicationWorks|StagingFilesAre|AdoptionPolicies|AHookRefresh|EveryStateWriter|ACloseRevalidates|TheBackfillNever -count=1
- source-coverage-integrity-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run SourceCoverage|UnreadableSpec|UnresolvedFollowups|FollowupProducerIsIncomplete|DroppedObligation|VeryLongLine|TechDebtCoverage|TechDebtReportSays|UnreadableNewerAttempt|UnreadableOlderAttempt|NoReadableAttempt|AnotherScopeDoesNotConcern|UnscopedUnreadable|UnreadableAttestation|UnreadableBundle|FindingsProducerIsIncomplete|BacklogAndTheProjectState|SpecItCouldNotRead|FailOverdueRefuses|AssessTechDebtStates|ReviewCheckCarries|FollowupParity|UnreadableLaterRejection|ReviewRecordRefusesToSupersede|ClosedScopeDoesNotRetain|Sniffing|DecoyInATruncated|ValidJSONWithoutAHeader|UnreadableAttemptOfTheSameSecond|OldMalformedAttempt|RequiredFields|FindingsLossInAnother|ComponentDiscoveryFinds|SkippedDirectoriesAndLinks|UnlistableSpecFolder|LegacySplitLayout|TruncatedHeaderIsALoss|AttentionCarriesTheFollowup|DebtClassificationSays|GovernanceConsumersDisclose|UnreadableLegacyLedger|InvalidElementWithoutAProducer|AttentionTextNames|CandidatesRefuse|UnknownDispositionAsUnresolved|ARejectionThatFailsItsDigest|RepeatedHeaderName|SniffingSeesARepeated|UnresolvableSpecEntry|FindingsLossOfTheRequested|DirectorySymlinkIsADeclared|ARoadmapOrTheProjectState|UnreadableLedgerDirectory|InvalidValuesIsALoss|NotPlannedOverAnIncomplete|ConsolidatedAssessmentDeclares|ProjectStateDoesNotCertify|HumanOutputsCarry|CaseFoldedHeaderNames|LostLaterBundleBlocks|ComparedAsInstants|TruncatedFindingLine|DanglingSpecFolderLink|DanglingReviewsRoot|LostRootForAbsence|InheritsTheLossOfItsMilestone|DoesNotCutTheTable|KeepsTheDebtReports|AttemptsAreReadInInstantOrder|ItemCutAnywhere|AttestationsAreOrderedByInstant|EmptyOpenFollowup|FutureSchemaIsUnsupported|UnreadablePersistedComponentState|WithoutASealedInstant|AnnotatedFollowupThatLost|PersistedComponentStateIsValidated|NoConsumerOrdersTheBundle|KeepsTheKindOfItsFailure|NamesItsScopeTwice|ReadsComponentsThroughTheValidating|ADiscoveryOfOneComponentKeeps|RepeatedAttemptFieldIsALoss -count=1
- index-generation-integrity-integration go test -race ./internal/pose ./internal/cli ./internal/mcpserver ./internal/scaffold/distpolicy -run IndexGeneration|TestIndexPublishes|TestIndexJSONReports|TestIndexVerify|TestIndexWithABlocked|TestRefreshDeliveryIndex|ReadsAPublishedGeneration|DoesNotInferADomain|TestIndexAnUnsyncedCommit|TestIndexWithAnotherPublication -count=1
- test-scripts-remove-their-temp-files-integration bash ../tests/hygiene/temp-cleanup.sh
- git-dispatch-snapshot-integrity-integration go test ./internal/pose ./internal/cli ./internal/mcpserver -run TestGitSnapshot|TestReviewDispatch -count=1

## Results
- Result: SUCCESS

## Execution Metadata
- Generated at (UTC): 2026-10-21T04:10:38Z
- Context: auto-validate
- Validation profile: strict
- Sequence for task/spec: 5
- Stable comparison hash: 62ef6fb6528d35aafbfea0aa2d1ac4121396c9c048ee6a208e8a8e667599e0a7

## Historical Comparison
- Previous execution: 2026-10-21T04:06:27Z
- Status: changed
- Stable field diffs:
- context: "Final native closeout with declared separate reviewer; 141 checks and 14 production mutations passed" -> "auto-validate"
- rules: "backend-go,security,documentation-style,delivery-surface" -> ""
- spec: "pose-git-and-dispatch-snapshot-integrity" -> ""
- workflow: "feature" -> ""

## Risks
- Actual final validation phase: 141/141 on d6ce7bc with isolated temporaries; the separate spec closeout and its constraints are recorded in the B07 implementation report

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
