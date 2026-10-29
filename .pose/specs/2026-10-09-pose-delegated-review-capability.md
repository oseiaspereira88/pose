---
slug: pose-delegated-review-capability
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-conclusions, pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger, pose-delegation-routing, pose-delegation-handoff, pose-delegated-review-round-protocol, pose-delegated-review-action-fulfillment, pose-delegation-mcp
priority: 3
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-review
---

# Spec: Delegated review as an adoptable capability

## 1. Intent

### Goal

Make delegated review a catalog capability that `pose init` offers with yes as the default, with documentation, a skill and agent examples, so a project adopts it like any other contract, and so a project that declines it still reaches delivery with an honest record.

### Business value

The maintainer asked for this to become POSE's standard way to hand reviews, attestations and similar work to a secondary agent, without the mechanisation turning into a rubber stamp.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

Amended on 2026-10-26 by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md): the capability is offered at init instead of opt-in only, runs through `pose delegate`, and when it is off a plan that requires `different-actor` degrades to a disclosed same-actor review (R1–R5 reworded, R8–R11 added).

### Constraints

Offered by `pose init`; an existing instance stays as it is until it adopts the capability; adopting it rewrites no earlier review record. The B17 delegated_review signed-run-proof barrier remains live: an older prefix-only approval can cease satisfying current verification after adoption. Requirements follow the maintainer's decisions recorded in the 2026-10-09 ADR and amended by the 2026-10-26 ADR. `mandatory-human` is never degraded.

## 2. Requirements

### Functional

- R1: `pose adopt delegated-review` shall turn on the run-backed independence rule, dated the day it is applied, and refuse without a review-capable agent, explicit executor-attested provenance, a loadable conclusions issuer and its exact attestation pin with project/audience binding. The B17 signed-run-proof barrier stays live for older approvals without rewriting their records.
- R2: The documentation shall describe the contract under `pose delegate`, the Codex and Claude Code agent examples, routing and its fallback steps, the degradation of R10 and every safeguard of both ADRs.
- R3: A `pose-delegate` skill shall tell an agent when to delegate a review and when a task, that it must not write the reviewer's prompt, that a rejection is answered by a change, not a rerun, and how to act on `next_actions`.
- R4: An end-to-end journey shall seal a fixture with a seeded defect, delegate its review to a fake agent, have the engine record the rejecting run, fix, run the next round on the new bundle and close with no manual step; a second journey escalates a person-required criterion to an action request; a third, with the capability off, closes with a disclosed same-actor review.
- R5: When a `pose delegate review` run passes every check — brief generated from the sealed bundle, disposable copy unchanged, run record intact and signed, `different-vendor` or `different-model` independence against the implementation recorded in the bundle, conclusion complete and accepted by the attestation preflight, no earlier rejection on an unchanged bundle — the engine shall record the attestation from the run with no further step; a rejecting conclusion shall be recorded and block the scope until the bundle changes.
- R6: Review policy shall declare which criteria require a person, and the engine shall turn each escalation — such a criterion, reviewers in conflict, a reviewer asking for a business decision, or the run budget exhausted without approval — into an action request with the question and options ready, answered as a confirmation or approval; nothing else waits on a person.
- R7: An `adjudication` or `smoke` run that passes the same checks shall record its verdict where its origin expects it (the adjudicated spec's decision record, the smoke target's evidence), under the same rules.

R5 and R7 are delivered by [`pose-delegated-review-conclusions`](2026-10-27-pose-delegated-review-conclusions.md) (phase B19, amendment of 2026-10-27 to the delegation ADR); this spec traces them to it and qualifies them in the integrated journeys of R4.
- R8: A fresh `pose init` shall offer delegation with yes as the default answer; on yes with supported CLIs found on the path it shall provision project-specific native executor signing, pin it only for attestations, write `delegation.json` with those agents, and enable `delegated_review: true` last, dated that day. A refusal, EOF, non-interactive init or zero supported CLIs shall record false without a delegation policy, key or pin. Existing governance is preserved even if its schema stamp is missing; partial publication retains recoverable artifacts and distinguishes unconfirmed durability.
- R9: `pose update` shall never turn the capability on; while it is off, it shall print one suggestion naming `pose adopt delegated-review`.
- R10: An explicit false `delegated_review` decision shall allow new plans requiring `different-actor` to accept `same-actor-separate-execution`, with `delegated-review-disabled` sealed into the plan and attestation and disclosed by verification and closeout. Legacy omission preserves prior behavior; mandatory-human and stronger protected baselines are unchanged. Disabling preserves pins and history and does not reconstruct proof or erase rejection or budget state.
- R11: While the capability is on and routing finds no available agent for a review after its fallback steps, the engine shall raise the escalation of R6 with the question ready, and shall not degrade as R10 does.

## 3. Technical Plan

### Affected areas

Catalog entry, `pose init` and `pose update` prompts, independence degradation in the review plan and its disclosure, docs page, skill, journey tests.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-capability.md
- created: .pose/changelogs/unreleased/pose-delegated-review-capability.md
- created: .pose/starts/pose-delegated-review-capability.json
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_assurance.go
- created: pose-mcp/internal/pose/delegated_review_adoption.go
- created: pose-mcp/internal/pose/delegated_review_adoption_test.go
- modified: pose-mcp/internal/cli/init.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/cli/init_delegation.go
- created: pose-mcp/internal/cli/delegated_review_capability_test.go
- created: .agents/skills/pose-delegate/SKILL.md
- created: pose-mcp/internal/scaffold/dist/.agents/skills/pose-delegate/SKILL.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/review-and-trust.md

- modified: .agents/skills/README.md
- modified: locales/pt-BR/.agents/skills/README.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/README.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/README.md
- modified: pose-mcp/internal/scaffold/scaffold.go
- modified: pose-mcp/internal/cli/skills_check_test.go
- created: locales/pt-BR/.agents/skills/pose-delegate/SKILL.md
- created: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-delegate/SKILL.md

- created: .pose/adr/2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
- created: .pose/actions/act-31f0849a7ec6712c.jsonl

- modified: pose-mcp/internal/scaffold/locale_coverage_test.go
- created: .pose/specs/2026-10-09-pose-delegated-review-capability.amendments.jsonl

- modified: pose-mcp/internal/pose/capability_catalog_test.go

- modified: pose-mcp/internal/pose/effective_governance.go

- modified: pose-mcp/internal/pose/review_policy_baseline.go
- modified: pose-mcp/internal/pose/delegation_routing.go
- modified: pose-mcp/internal/mcpserver/delegate_tools.go
- modified: pose-mcp/internal/mcpserver/delegate_tools_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- created: .pose/reports/delegation-adoption-review-round1.md

### Delivery targets

- capability:delegated-review module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/delegated_review_adoption.go

## 4. Tasks

- [ ] Catalog entry
- [ ] Init offer and update suggestion
- [ ] Degradation when off, escalation when no agent is available
- [ ] Docs and agent examples
- [ ] Skill
- [ ] Journeys

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

The [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md), accepted on 2026-10-26, amends it: delegation is offered by `pose init` with yes as the default instead of being opt-in only; with the capability off, `different-actor` degrades to a disclosed `same-actor-separate-execution` so one agent of one vendor can reach delivery; the implementation identity comes from the sealed bundle; runs go through `pose delegate`.

The [adoption ADR](../adr/2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md) clarifies absence versus refusal, machine-only signing bootstrap and recoverable publication. Maintainer decision `act-31f0849a7ec6712c` selected the live B17 barrier on 2026-10-29; this spec promises immutable historical records, not grandfathered proof exemptions.

### Decision D1
- Basis: R10, R11
- Minimal option: keep `different-actor` mandatory whether or not the capability is on
- Selected option: degrade to a disclosed same-actor review when the capability is off; escalate, never degrade, when it is on and no agent is available
- Rationale: an instance that declined delegation has no second actor to name, and requiring one only invites a declared reviewer nobody verified; an instance that adopted it has promised independence, so a missing agent is a question for a person, not a silent downgrade
- Consequences: the review record of a declining instance always shows the limitation
- Falsifier: a closeout with the capability on that recorded `same-actor-separate-execution` for a `different-actor` plan without an escalation

### Decision D2
- Basis: R1, R8, R9, R10
- Minimal option: treat a missing delegated_review key as a new refusal
- Selected option: preserve the legacy policy when the key is absent; an explicit false decision on a new init or off adoption allows disclosed same-actor execution for new bundles. A protected baseline and mandatory-human remain stronger. An existing init never asks again or rewrites that decision. Adoption and off dates are recorded; existing sealed plans and records are immutable, while the B17 live run-proof trust bar still re-evaluates an older approval after adoption
- Rationale: the accepted ADR changes new adoption, not history or an existing instance without a decision
- Consequences: capture delegated-review-disabled in the sealed plan and derived record/closeout assurance; omission keeps previous digests unchanged
- Falsifier: upgrading alone relaxes an old policy or a mandatory-human requirement

### Decision D3
- Basis: R1, R5, R8
- Minimal option: write agent examples and enable the flag without signing prerequisites
- Selected option: on an accepted fresh interactive init, detect supported CLIs without invoking them, use the existing native issuer and pin machinery for a project-specific local executor issuer outside the repository, preserve authority project/audience, configure executor-attested provenance and conclusion issuer, and only then enable the flag. No human identity or human-authority pin is created. If no supported CLI is found, leave delegation explicitly disabled with a diagnostic. Non-interactive init never enables it. Explicit adopt of an existing policy validates agents and the existing trust prerequisites rather than inventing an issuer
- Rationale: an enabled capability must be able to produce the signed run proof B19 requires; an empty agent policy is not adoption
- Consequences: local keys are never versioned; refusal and non-interactive operation write no delegation policy; update only suggests. Existing files are never overwritten by the init offer. The decision is published last under compare-and-swap and configuration revalidation; partial setup retains artifacts for recovery and reports published-but-unsynced state distinctly
- Falsifier: an init claims enabled delegation without a usable agent or trust configuration, overwrites policy, or records a human

Review context: knowledge:delegation-routing-round3; knowledge:delegated-conclusions-round4; knowledge:delegation-mcp-round5. B19 conclusions and B20 fulfillment are already done. B21 qualifies their composition; no new phase checkpoint is introduced. The 2026-10-26 ADR's init offer supersedes the older batch-10 wording about off-by-default interactive opt-in. Disabling the capability preserves runs, attempts, rejections, issuer pins and earlier approvals; it does not cancel an execution or restore a spent budget.

## 6. Validation

### Strategy

The journey tests fail before the capability exists and pass after. An interactive init fixture writes `delegation.json` from fake CLIs on the path; a non-interactive init records the capability off; `pose update` suggests without enabling; with the capability off a `different-actor` plan closes with the disclosed limitation and a `mandatory-human` plan does not; with it on and every agent unavailable, the escalation is raised.

### Test plan (required, high criticality)

Run Go checks sequentially with GOMAXPROCS=2 and GOFLAGS=-p=1. Use temporary fake CLIs, isolated issuer key directories and no real vendor execution.

| Scenario | Command | Expected evidence |
|---|---|---|
| Catalog adoption, missing/malformed agent policy, on/off date and replay | `go -C pose-mcp test ./internal/pose -run DelegatedReviewCapability -count=1` | Refusal writes nothing; valid adoption is dated and idempotent; off preserves history |
| Explicit off versus legacy omission, captured limitation, human and protected baseline | `go -C pose-mcp test ./internal/pose -run DelegatedReviewCapability -count=1` | Only explicit off relaxes different-actor; mandatory-human/protected baseline remain required; old plan digests unchanged |
| Interactive default yes/no, no CLI, non-interactive and repeated init | `go -C pose-mcp test ./internal/cli -run 'DelegationInit|DelegatedReviewCapability' -count=1` | Only an accepted fresh offer creates policy and issuer; refusals/headless record off; no vendor starts; existing instance unchanged |
| Update preview/apply and capability-off suggestion | `go -C pose-mcp test ./internal/cli -run 'DelegatedReviewCapability|UpdatePreviewParity' -count=1` | One suggestion, never activation; preview and apply agree |
| Review rejection, changed bundle round, automatic signed conclusion, person and unavailable-agent exceptions; existing adjudication/smoke destinations | `go -C pose-mcp test ./internal/pose ./internal/cli ./internal/mcpserver -run 'DelegatedReviewCapability|DelegationInit|AutoConclude|DelegatedReviewRequestPerKind|ActionFulfil' -count=1` | No manual attestation/conclude step; rejection remains restrictive; required-person and unavailable-agent requests carry authority/options; task runs never count |
| Registered composition | `pose validate --strict --json-out .pose/results/delivery-validation.json` | All registered checks pass with current provenance, including new capability integration check |
| Final source and delivery contracts | `pose artifact-check --spec pose-delegated-review-capability --strict` and `pose surface-check --spec pose-delegated-review-capability --strict` | Attributed paths reconcile and production entrypoint has current integration evidence |

Additional required checks: full Go suite and vet; focused race tests for delegation/adoption; Windows/Darwin builds; secret scan; distributed manual and skill parity. Review, native closeout and CI must pass before declaring done. Failures and retries are preserved rather than silently replaced in the review report.

### Requirement trace

- R1 [satisfied] Exact trust validation, dated idempotent adoption, protected floor and the retained live barrier: test:TestDelegatedReviewCapabilityValidatesTheExactExecutor test:TestDelegatedReviewCapabilityAdoptsAndExplicitlyTurnsOff test:TestDelegatedReviewCapabilityProtectsTheAdoptedFloor test:TestRunProvenanceLiveOptInDoesNotGrandfatherPrefix evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R2 [satisfied] The operating manuals, native help and docs-site describe routing, signing, recovery and limitations; distributed locale/skill guards cover parity: test:TestLocaleCoverage test:TestSkillsCheckDiscoveryAndBoundedWorkflowFixture file:docs-site/docs/review-and-trust.md evidence:integration check:pose-mcp/go/test
- R3 [satisfied] The distributed pose-delegate skill distinguishes governed review from task, native briefs from labelled notes, remediation from reruns and authorized next actions: file:.agents/skills/pose-delegate/SKILL.md test:TestClaudeSkillLinksMatchAgentsSkills test:TestSkillsCheckDiscoveryAndBoundedWorkflowFixture evidence:integration check:pose-mcp/go/test
- R4 [satisfied] Native CLI journeys reject a source-seeded defect, preserve rejection, accept the changed bundle automatically and close; person/MCP and explicit-off journeys cover the exceptions: test:TestDelegatedReviewCapabilityRejectFixApproveAndClose test:TestDelegatedReviewCapabilityEscalatesThroughMCP test:TestDelegatedReviewCapabilityOffClosesWithDisclosure evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R5 [satisfied] B19 conclusions compose through the same verified writer and rejection history: test:TestConcludeReviewRunRecordsTheAttestation test:TestConcludeReviewRunRecordsARejection test:TestDelegatedReviewCapabilityRejectFixApproveAndClose evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R6 [satisfied] Person, unavailable/insufficiently independent agents and the B20 business/budget/conflict paths use native authority-bound decisions: test:TestDelegatedReviewCapabilityPersonAndSameModelEscalate test:TestDelegatedReviewCapabilityEscalatesThroughMCP test:TestDelegatedReviewRequestEscalatesToAPerson evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R7 [satisfied] B19/B20 per-kind destinations remain distinct and runner-independent: test:TestDelegatedReviewRequestPerKind evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R8 [satisfied] Accepted, refused, headless, missing CLI, repeated/missing-stamp, linked governance and partial-publication cases use the public init and native issuer machinery: test:TestDelegationInitAcceptedOfferCreatesOnlyExecutorTrust test:TestDelegationInitRefusalAndHeadlessCreateNoSigningArtifacts test:TestDelegationInitDoesNotBootstrapOverExistingGovernance test:TestDelegationInitPolicyPublicationFailureRetainsRecoverableState test:TestDelegatedReviewCapabilityValidatesTheCapturedConfiguration evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R9 [satisfied] Update emits one suggestion while preserving disabled policy; adoption preview is mutation-free: test:TestDelegatedReviewCapabilityUpdateOnlySuggests test:TestDelegatedReviewCapabilityAdoptPreviewAndOff evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R10 [satisfied] Only explicit refusal relaxes new eligible plans, with sealed/read-time disclosure and unchanged protected/human floors: test:TestDelegatedReviewCapabilityDisclosesOnlyAnExplicitRefusal test:TestDelegatedReviewCapabilityOffClosesWithDisclosure test:TestDelegatedReviewCapabilityDoesNotRelaxAProtectedBaseline evidence:integration check:pose-mcp/go/delegated-review-capability-integration
- R11 [satisfied] Unavailable agents open one decision without a run; routing skips same-model candidates and tries independent ones first: test:TestDelegatedReviewCapabilityUnavailableAgentEscalatesWithoutSpawn test:TestDelegatedReviewCapabilityRoutesPastASameModelAgent test:TestDelegatedReviewCapabilityEscalatesThroughMCP evidence:integration check:pose-mcp/go/delegated-review-capability-integration

## 7. Final Report

### Delivered scope

Implementation and focused positive/negative journeys are prepared in the adoption milestone of roadmap `delegation`. Final validation and implementation review remain pending. Maintainer decision `act-31f0849a7ec6712c` selected the live B17 barrier; the adoption ADR records activation/recovery guarantees and the selected historical contract.

### Residual risks

A trusted local executor can emit a complete canned judgment. Native proof authenticates the executor declaration and bound artifacts, not substantive review cognition or a vendor receipt; the adversarial disclosure test guards that boundary. The technical-debt assessment has zero uncovered markers but reports incomplete coverage for two pre-existing legacy spec sources; it is not evidence of global completeness.

### Follow-ups
