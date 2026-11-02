---
slug: pose-milestone-review-proportional
status: done
created_at: 2026-11-02
supersedes:
depends_on: pose-milestone-closes-when-reached
remediates:
priority: 1
components: pose-mcp
task_type: feature
delivers: capability:milestone-review-proportional
completed_at: 2026-11-02
---

# Spec: Milestone review is proportional to its members

## 1. Intent

### Goal

A milestone of one member is approved when its spec is, a milestone of several keeps its review, and an old roadmap can say why its milestones were not reviewed.

### Business value

Reviewing the 42 reached milestones took a night and 145 thousand lines of bundles, and 100 of the 133 milestones with members have one member, whose review repeats the review of its spec. The ADR `.pose/adr/2026-11-02-a-milestone-of-one-member-is-that-member.md` records why no condition is attached.

### Constraints

- `review verify`, closeout, the review of a roadmap and the federated proof give one answer.
- A negative review recorded on a milestone stands.
- Edges still bind: a milestone inherits a review, not the acceptance of what it depends on.
- Closed scopes keep their approvals.

### Non-goals

- Changing what a milestone of several members owes.
- Backfilling the exit gate of specs already closed.

## 2. Requirements

### Functional

- R1: Under a review policy with bundles, a milestone with exactly one local member, closed and approved, whose federated acceptance holds, shall read approved and closed with no bundle of its own, and say where the approval comes from.
- R2: A negative review recorded on such a milestone, in a bundle or a legacy attempt, or an attempt history that cannot be read, shall keep it unapproved, and a recorded approval shall stay as it is.
- R3: A milestone with an external member or with several members shall owe its review as before.
- R4: The review plan of an open spec that is the only member of a milestone of a live roadmap shall carry a required judgment criterion holding the exit gate of that milestone, one criterion per gate.
- R5: A roadmap that declares `unreviewed_milestones` with a reason shall treat a milestone whose members are all terminal as standing, in verification and closeout, one member or many, with the reason sealed and readable in the bundle of the roadmap, whose own review stays required and is what an external consumer holds.
- R6: The review of a roadmap shall seal over inherited and declared milestones, and `pose state --attention` shall not list them as reached, while a milestone that does not stand stays listed.

### Non-functional

- Verification of an inherited milestone costs the verification of its spec, which the operation reuses.

### Security

- This drops a review, so it comes with an ADR. A milestone gains approval only from a closed, approved spec or from a reason the roadmap declares and its review seals.

### Compatibility

- Bundles of milestones keep working. A spec in progress that is the only member of a milestone gains one criterion, so a bundle already sealed for it reads superseded and is sealed again. A policy without bundles is untouched.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`, `internal/pose/review_plan.go`, `internal/pose/roadmaps.go`, `internal/pose/obligation_projection.go`, the closeout skill.

### Artifacts

- created: .pose/specs/2026-11-02-pose-milestone-review-proportional.md
- created: .pose/starts/pose-milestone-review-proportional.json
- created: .pose/changelogs/unreleased/pose-milestone-review-proportional.md
- created: .pose/adr/2026-11-02-a-milestone-of-one-member-is-that-member.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/roadmaps.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/federated_acceptance.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- created: pose-mcp/internal/pose/milestone_review_proportional_test.go
- modified: pose-mcp/internal/pose/federated_acceptance_test.go
- modified: pose-mcp/internal/pose/review_approval_follows_children_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/milestone_closes_when_reached_test.go

### Delivery targets

- capability:milestone-review-proportional module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`review verify` of a one-member milestone reports the bundle of its spec. Roadmap frontmatter gains `unreviewed_milestones`.

## 4. Tasks

- [x] Inherit in the one verification every reader uses
- [x] Carry the exit gate into the review of the spec
- [x] Let a roadmap declare unreviewed milestones
- [x] Measure every milestone of this repository, old reader against new
- [x] Update the closeout skill
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3
- Minimal option: reviewers attest one-member milestones in bulk
- Selected option: no review, as the ADR records
- Rationale: the review repeated its spec, and a condition on a declared check would have passed over the defect that argued for it
- Consequences: the bundle of the spec answers for the milestone
- Falsifier: a defect found in a one-member milestone that the review of its spec could not have seen, with its exit gate in hand

### Decision D2
- Basis: R4
- Minimal option: leave the gate unjudged
- Selected option: a criterion in the plan of an open spec, none for a done one
- Rationale: the gate is the one thing the milestone adds, and a done spec keeps the plan it was sealed with
- Consequences: a spec in progress owes one more judgment
- Falsifier: a gate that a reviewer of the spec cannot judge

### Decision D3
- Basis: R5
- Minimal option: review every old multi-member milestone
- Selected option: a reason declared in the roadmap and sealed by its review
- Rationale: an old roadmap needs an honest record, not a retroactive review
- Consequences: a roadmap can skip reviews by writing a reason, which its reviewer reads
- Falsifier: a declared reason used to hide a milestone that is not reached

## 6. Validation

### Strategy

Integration tests over real bundles for each reader, seen failing first, and every milestone of this repository read by the old and the new engine: nothing approved may fall.

### Requirement trace

- R1 [satisfied] capability:milestone-review-proportional evidence:integration check:milestone-review-proportional-integration test:TestOneMemberMilestoneInheritsItsSpec test:TestAMilestoneDoesNotInheritAnOpenOrUnapprovedSpec test:TestAnApprovedOpenSpecIsNotInherited test:TestAPolicyWithoutBundlesKeepsItsOwnAnswer test:TestAMilestoneInheritsNoAcceptance test:TestCloseOfAnInheritingMilestonesSpecNamesTheRoadmap report:.pose/results/delivery-validation.json commit:117c1c0068b7851841228686f6e0445640c4da08
- R2 [satisfied] check:milestone-review-proportional-integration test:TestANegativeReviewOfAMilestoneStands test:TestARecordedMilestoneApprovalStays test:TestARecordedLegacyRefusalStands report:.pose/results/delivery-validation.json commit:117c1c0068b7851841228686f6e0445640c4da08
- R3 [satisfied] check:milestone-review-proportional-integration test:TestOtherMilestonesOweTheirReview report:.pose/results/delivery-validation.json commit:117c1c0068b7851841228686f6e0445640c4da08
- R4 [satisfied] check:milestone-review-proportional-integration test:TestTheSpecAnswersForTheExitGate test:TestEachExitGateHasItsOwnCriterion report:.pose/results/delivery-validation.json commit:117c1c0068b7851841228686f6e0445640c4da08
- R5 [satisfied] check:milestone-review-proportional-integration test:TestARoadmapDeclaresUnreviewedMilestones test:TestTheReasonIsSealedByTheRoadmap test:TestARetainedRoadmapDoesNotCoverADeclarationItNeverRead test:TestADeclarationRescuesATerminalSingleton test:TestAnExternalConsumerHoldsTheRoadmapOfADeclaredMilestone report:.pose/results/delivery-validation.json commit:117c1c0068b7851841228686f6e0445640c4da08
- R6 [satisfied] check:milestone-review-proportional-integration test:TestRoadmapSealsOverInheritedMilestones test:TestRoadmapSealsOverADeclaredMilestone test:TestAttentionKeepsWhatDoesNotStand test:TestAttentionKeepsLegacyRefusals test:TestALegacyApprovalDoesNotEraseABundleRefusal test:TestALaterLegacyApprovalReplacesALegacyRefusal test:TestReachedListsAnUnapprovedSingleton report:.pose/results/delivery-validation.json commit:117c1c0068b7851841228686f6e0445640c4da08

## 7. Final Report

### Delivered scope

A milestone of one local member has no review of its own: it stands on the closed, approved review of its spec, through the one verification that `review verify`, closeout, the review of a roadmap and the federated proof read, and `approval_origin` says so. It inherits a review and nothing else: its edges still have to hold, and a refusal recorded on it, in a bundle or a legacy attempt, stands. An open spec answers for the exit gate of its milestone. A roadmap can declare why old milestones stand unreviewed, and its bundle seals that reason, so a retained approval does not cover a declaration it never read. Of 171 milestone and roadmap scopes here, none fell and 16 rose, each judged correct by the reviewer. Independently reviewed by Codex in three rounds, which found ten blockers between them, from a failed acceptance erased by inheritance to a legacy approval hiding a recorded refusal.

### Residual risks

The review of a one-member milestone that happened to come later is gone, and the review of the roadmap is where a second look now belongs. The exit gate is a judgment, which a reviewer can answer as not applicable with a rationale. `pose state --attention` reads records and does not verify, so its list can differ from a verification on a stale approval. A policy without bundles keeps its own answer.

### Follow-ups

- [open] F1: Add a representative cost measure of roadmap preparation, with distinct members and gates: reviewed sizes took up to two seconds and no budget exists (review R2-F2). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F2: Add a milestone-gate case to the adversarial corpus and a journey test of the gate criterion on a fresh install (review R2-F3). (owner:@oseiaspereira88 crit:low review:2026-11-27)
