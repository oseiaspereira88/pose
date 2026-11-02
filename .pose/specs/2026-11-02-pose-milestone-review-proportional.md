---
slug: pose-milestone-review-proportional
status: in-progress
created_at: 2026-11-02
supersedes:
depends_on: pose-milestone-closes-when-reached
remediates:
priority: 1
components: pose-mcp
task_type: feature
delivers: capability:milestone-review-proportional
completed_at:
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
- Edges still bind: federated acceptance of the milestone is untouched.
- Closed scopes keep their approvals.

### Non-goals

- Changing what a milestone of several members owes.
- Backfilling the exit gate of specs already closed.

## 2. Requirements

### Functional

- R1: A milestone with exactly one local member, closed and approved, shall read approved and closed with no bundle of its own, and say the approval is inherited.
- R2: A recorded negative review of such a milestone shall keep it unapproved, and a recorded approval shall stay as it is.
- R3: A milestone with an external member or with several members shall owe its review as before.
- R4: The review plan of an open spec that is the only member of a milestone shall carry a required judgment criterion holding the exit gate of that milestone.
- R5: A roadmap that declares `unreviewed_milestones` with a reason shall treat a milestone whose members are all terminal as standing, and its own review shall stay required.
- R6: The review of a roadmap shall seal over inherited and declared milestones, and `pose state --attention` shall not list them as reached.

### Non-functional

- Verification of an inherited milestone costs the verification of its spec, which the operation reuses.

### Security

- This drops a review, so it comes with an ADR. A milestone gains approval only from a closed, approved spec or from a reason the roadmap declares and its review seals.

### Compatibility

- Bundles of milestones keep working. A spec in progress that is the only member of a milestone gains one criterion.

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
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- created: pose-mcp/internal/pose/milestone_review_proportional_test.go

### Delivery targets

- capability:milestone-review-proportional module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`review verify` of a one-member milestone reports the bundle of its spec. Roadmap frontmatter gains `unreviewed_milestones`.

## 4. Tasks

- [ ] Inherit in the one verification every reader uses
- [ ] Carry the exit gate into the review of the spec
- [ ] Let a roadmap declare unreviewed milestones
- [ ] Measure every milestone of this repository, old reader against new
- [ ] Update the closeout skill
- [ ] Independent review and closeout

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

- R1 [planned] capability:milestone-review-proportional evidence:integration check:milestone-review-proportional-integration test:TestOneMemberMilestoneInheritsItsSpec
- R2 [planned] check:milestone-review-proportional-integration test:TestANegativeReviewOfAMilestoneStands test:TestARecordedMilestoneApprovalStays
- R3 [planned] check:milestone-review-proportional-integration test:TestOtherMilestonesOweTheirReview
- R4 [planned] check:milestone-review-proportional-integration test:TestTheSpecAnswersForTheExitGate
- R5 [planned] check:milestone-review-proportional-integration test:TestARoadmapDeclaresUnreviewedMilestones
- R6 [planned] check:milestone-review-proportional-integration test:TestRoadmapSealsOverInheritedMilestones

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
