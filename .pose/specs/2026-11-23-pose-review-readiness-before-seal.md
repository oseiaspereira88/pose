---
slug: pose-review-readiness-before-seal
status: done
created_at: 2026-11-23
completed_at: 2026-11-23
supersedes:
depends_on: 
remediates:
priority: 1
components: pose-mcp
task_type: feature
surface: standard
delivers: capability:review-readiness-before-seal
---

# Spec: pose-review-readiness-before-seal

## 1. Intent

### Goal
Refuse at seal, and let the reviewer check inside its run, every condition the conclusion preflight and `pose close` would otherwise refuse after a full review round.

### Business value
Closing `pose-validate-carries-unchanged-checks` took 23 review rounds. About seven were lost to conditions known before the reviewer started: a missing delivery declaration that `close` refused only after approval, a decision with no requirement basis, a finding id with a round prefix, `prior_findings` naming findings no concluded round holds open, criterion evidence outside the criterion's component, and a note file over 16 KiB. Each cost a seal, a delegated run and about 40 minutes.

### Constraints
The checks reuse the existing preflight and closeout predicates; no rule is duplicated. A seal refusal names the rule and the fix. Nothing weakens a gate: what is refused later is refused earlier, never accepted.

### Non-goals
Capping review rounds, changing review independence, or judging the subject itself.

## 2. Requirements

### Functional
- R1: `pose review bundle --seal` shall refuse a spec scope whose attributed changes touch a delivery root without a delivery declaration, or a governed path its artifacts do not declare, with the same findings `pose close` refuses (`undeclared-delivery`, `undeclared`).
- R2: When the plan asks for structural mappings, the seal shall refuse a spec scope with a decision or assumption that reaches no requirement or constraint, the rule the conclusion preflight applies to a mapping basis.
- R3: A new `pose review draft-check <run|bundle> --draft <file>` shall run inside the review sandbox without writing. It reports every reason the conclusion preflight would refuse that draft: unreadable block, prefixed new finding ids, `prior_findings` outside the open findings, criterion evidence outside its component, undeclared decisions in mappings. It exits non-zero when any applies.
- R4: The review brief shall instruct the reviewer to run `draft-check` on its draft before finishing, and list the open prior findings it must account for.
- R5: `pose delegate` shall refuse a note file over the adapter limit with a message naming the note, its size and the limit, instead of the generic adapter-file error.
- R6: When the instance has a conclusions issuer but the run is not concluded automatically, `pose delegate review` shall print the exact `pose review conclude <run>` command as the next step.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` (CLI and engine), the manual and its embedded copy.

### Artifacts
- created: .pose/specs/2026-11-23-pose-review-readiness-before-seal.md
- created: .pose/adr/2026-11-23-review-readiness-before-seal.md
- created: pose-mcp/internal/pose/review_seal_readiness.go
- created: pose-mcp/internal/pose/review_draft_check.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- modified: pose-mcp/internal/pose/review_brief_conclusion_rules_test.go
- modified: pose-mcp/internal/pose/review_brief_preflight_tie_test.go
- modified: pose-mcp/internal/pose/live_policy_reads_test.go
- modified: pose-mcp/internal/pose/review_structure_test.go
- modified: pose-mcp/internal/pose/testdata/review_brief_template_digests.json
- created: pose-mcp/internal/cli/review_draft_check.go
- created: pose-mcp/internal/cli/review_draft_check_test.go
- created: pose-mcp/internal/cli/review_seal_readiness_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_conclude.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:review-readiness-before-seal module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_draft_check.go

### API/contract changes
New command `pose review draft-check`; new environment variables `POSE_REVIEW_DRAFT_CONTEXT` and `POSE_REVIEW_CLI` set for delegated review runs; new brief template `review-brief/8`; `review bundle --seal` refuses two more conditions for new seals.

### Data/storage changes
The dispatch writes `pose-review-draft-context.json` to the run's private temporary directory; nothing is written to the repository.

### Technical risks
A draft check that drifted from the conclusion would mislead reviewers; it calls the same functions, and a test compares its reasons with the conclusion's refusal. Residual limits L1 and L2 are stated in the ADR.

## 4. Tasks

### Planning
- [ ] Confirm the requirements against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R3, R4
The draft check reuses the conclusion's parser, round accounting and attestation content checks, extracted as `attestationContentBlockers`, so it cannot accept what the conclusion refuses for content. It leaves out only the run's signature and identity, which do not exist before the run ends. See ADR `2026-11-23-review-readiness-before-seal`.

### Decision D2
- Basis: R1, R2
The seal applies only the readiness rules whose inputs exist before the review: the delivery and artifact declarations and the decision basis, the latter only when a required criterion asks for structural mappings. Evidence and trace gates of `pose close` stay at close. A bundle already sealed is returned without being judged again.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-23: implemented R1–R6 with a regression test each, shown failing with its fix mutated out (seal refusal, decision basis, draft-check content, dispatch context, oversized note, conclude hint). `go test ./...` in `pose-mcp` passes.
- 2026-11-23: review round 1 (rrn-fa6a6e8e1df0a0ba) requested changes. N1: the decision-basis refusal applied to plans with structural facts but no criterion asking for their mapping; it now requires a required `RequiresStructuralMapping` criterion. N2: `draft-check` rejected run ids; a run id now selects its bundle. N3: `live_policy_reads_test.go` was not declared; it is, and the seal now also refuses the `undeclared` artifact finding, which the round showed is known before the review too. The reviewer could not run `draft-check` because its shell resolved the installed `pose`; the dispatch now sets `POSE_REVIEW_CLI` to the dispatching engine and `review-brief/8` (not yet published) names it. Each fix has a test that fails with it mutated out.
- 2026-11-23: review round 2 (rrn-9d5816c780952f13) ran `draft-check` through `POSE_REVIEW_CLI` on its own draft, which passed, and found N4 and N5; its cleanup was incomplete because a broad test was still running, so it could not be concluded. N4: a draft over 4 MiB was judged by its prefix; drafts over the limit are now refused from a file or standard input. N5: naming the run's own bundle inside the sealed checkout ignored the dispatch context; the context now serves the run's bundle whether it is named or not, and another reference names the context's bundle when it cannot be read. Both fixes have tests that fail with them mutated out.


### Results summary
The full `pose-mcp` suite passes. The canonical validation result is refreshed with `pose validate --carry` after the implementation commit.


### Requirement trace
- R1 [satisfied] test:TestSealRefusesAnUndeclaredDelivery
- R2 [satisfied] test:TestSealReadinessRefusesADecisionWithoutBasis
- R3 [satisfied] test:TestReviewDraftCheckMatchesTheConclusionStep
- R4 [satisfied] test:TestTheCurrentBriefAsksForTheDraftCheck
- R5 [satisfied] test:TestDelegateNamesTheNoteLimitAndTheConclusionStep
- R6 [satisfied] test:TestDelegateNamesTheNoteLimitAndTheConclusionStep


### Known gaps
The draft check cannot run the signature and identity checks, which need the finished run; the conclusion step still runs them.


## 7. Final Report

### Delivered scope
The seal of a new spec bundle refuses an undeclared delivery or artifact and, when the plan asks for structural mappings, a decision basis that reaches no requirement. `pose review draft-check` reports what the conclusion step would refuse, inside the run, through a context the dispatch writes. The brief `review-brief/8` asks for it. `pose delegate` names an oversized note and the conclude command of a run nothing concluded.


### Files and modules changed
`pose-mcp` (CLI and engine), the manual in both locales and its embedded copies, the ADR `2026-11-23-review-readiness-before-seal`.


### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.


### Residual risks
L1 and L2 in the ADR: the draft check judges against the dispatch's context, and the seal reads the committed index.


### Follow-ups
None.


