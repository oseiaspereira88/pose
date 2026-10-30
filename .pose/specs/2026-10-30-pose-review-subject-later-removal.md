---
slug: pose-review-subject-later-removal
status: in-progress
created_at: 2026-10-30
supersedes:
depends_on: pose-delivery-later-removal
remediates:
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:review-subject-later-removal
---

# Spec: A milestone review can be sealed after its specs closed and a later spec removed their files

## 1. Intent

### Goal

Seal the review bundle of a milestone whose member specs are done and whose files a later spec removed, with the same answers `review verify` and `check --strict` already give for those specs.

### Business value

Closing the roadmaps `delegated-review` and `delegation` needs a sealed `milestone-integration@1` bundle per milestone. All eighteen refuse: (a) six of their specs declare `pose-mcp/internal/cli/review_dispatch.go`, which `pose-delivery-later-removal` removed, and the review subject still tries to read it — the delivery graph accepts the removal, the review subject does not; (b) the seal requires each member spec to have a sealed bundle equal to the one prepared now, while a done spec's later commits (its closeout, the index, a later removal) change what is prepared — `review verify` and the closeout gate retain the approved review of a done scope, the milestone seal does not.

### Constraints

- A removal is accepted only as the delivery graph accepts it (`pose-delivery-later-removal` R1–R3); nothing here widens that rule.
- A retained child review is accepted only where the closeout gate already retains it: a done scope, the latest readable attestation approved, the attestation history complete.

### Non-goals

- Milestones whose members are not done, or roadmaps' own review rules.
- Re-reviewing or re-sealing closed specs.

## 2. Requirements

### Functional

- R1: When a review subject path is absent from the tree, and the delivery-integrity graph has, for every selected spec that observes the path, a removal of it by another spec that supersedes that very spec, the subject entry shall be digested from those witnesses (removing spec and deleting commit per observer) instead of failing as unreadable.
- R2: An absent path shall still refuse the bundle with the unreadable-path blocker when any observing spec lacks such a witness — no witness, a witness for another path or another spec, a refused removal, or the spec's own removal — whatever the order of the selected change sets.
- R3: Sealing a milestone (or roadmap) bundle shall accept a done child scope whose sealed bundle no longer equals the prepared one when the child's own review verification retains its approved review (closed, fresh, approved — lost history, a later negative review and a changed federated manifest excluded), recording the retained bundle as the child; otherwise the child shall still block, with the verification's cause.
- R4: The eighteen milestone bundles of the roadmaps `delegated-review` and `delegation` shall prepare with no blocker under these rules; sealing them follows the engine specs for SR-R1-05, SR-R1-06 and SR-R1-07 (maintainer's direction, 2026-10-30).

### Non-functional

- No extra Git read: the witness comes from the graph the bundle already builds.

### Security

- None beyond the existing removal and retention rules.

### Compatibility

- Bundles of subjects without removed paths and of milestones whose children match keep their digests.

## 3. Technical Plan

### Affected areas

Review bundle subject and child resolution in `internal/pose/review_bundle.go`.

### Artifacts

- created: .pose/specs/2026-10-30-pose-review-subject-later-removal.md
- created: .pose/starts/pose-review-subject-later-removal.json
- created: .pose/changelogs/unreleased/pose-review-subject-later-removal.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_subject_later_removal_test.go
- modified: pose-mcp/internal/pose/delegated_review_adoption_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- created: .pose/specs/2026-10-30-pose-review-subject-later-removal.amendments.jsonl
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-subject-later-removal module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_closeout.go

### API/contract changes

A review subject entry may carry a removal digest; a milestone bundle's child may be a retained closed review.

## 4. Tasks

- [x] Removal witness in the review subject
- [x] Retained children in milestone seals
- [x] Prepare the eighteen milestone bundles without blockers
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R3
- Minimal option: re-seal and re-attest every closed member spec before sealing a milestone
- Selected option: reuse the answers the engine already gives — the delivery graph's removal witness and the closeout gate's retained review
- Rationale: the closed specs' reviews are already fresh and approved for `review verify` and `check --strict`; a second, stricter answer at the milestone seal forces re-reviews that judge nothing new (maintainer's choice, 2026-10-30)
- Consequences: a milestone bundle may name a child bundle older than the child's current preparation
- Falsifier: a milestone seal that accepts a child whose closeout gate is not fresh and approved

## 6. Validation

### Strategy

Fixtures: a spec whose declared file a later spec removed seals its bundle with a removal entry; without the witness, or with a refused one, it refuses. A milestone whose done child changed after closeout seals with the retained child; a not-done child or a child with a rejected latest review blocks. Then the eighteen real milestones seal. Each rule shown failing on revert.

### Requirement trace

- R1 [satisfied] check:review-subject-later-removal-integration test:TestReviewSubjectReadsALaterRemovalFromItsWitness test:TestMilestoneSubjectRemovalHoldsForEveryMember report:.pose/results/delivery-validation.json commit:814d1c1032d5501b574c8ceda715d7150896b2c4 — An absent subject path digests from the removals the delivery graph accepted for every selected spec observing the path it reads, keyed by that path before change-set reduction.
- R2 [satisfied] check:review-subject-later-removal-integration test:TestReviewSubjectReadsALaterRemovalFromItsWitness test:TestReviewSubjectRemovalHoldsForEveryObserver test:TestMilestoneSubjectRemovalHoldsForEveryMember report:.pose/results/delivery-validation.json commit:814d1c1032d5501b574c8ceda715d7150896b2c4 — No witness, another path's or spec's, a refused removal, a spec's own removal, or any observer lacking one keeps the unreadable-path blocker, in every set order, action and containment.
- R3 [satisfied] check:review-subject-later-removal-integration test:TestMilestoneSealRetainsADoneChildsApprovedReview test:TestRetainedChildFollowsItsVerification report:.pose/results/delivery-validation.json commit:814d1c1032d5501b574c8ceda715d7150896b2c4 — A done child whose preparation changed is accepted through the bundle its own review verification retains; not-done children, later negative reviews and revoked federated trust keep the blocker.
- R4 [satisfied] check:review-subject-later-removal-integration report:.pose/results/delivery-validation.json commit:814d1c1032d5501b574c8ceda715d7150896b2c4 — All eighteen milestones of `delegated-review` and `delegation` prepare with zero blockers (independently reproduced by the reviewer); their seals wait on F1–F3.

## 7. Final Report

### Delivered scope

A review subject reads a path a later spec removed through the removal the delivery graph accepted for each observing spec, and a milestone seal accepts a done member through the review its own verification retains. Independently reviewed by Codex over four rounds (reports kept outside the checkout); 11 of 12 mutations killed, the survivor adjudicated equivalent by the reviewer.

### Residual risks

A milestone's integration approval certifies the child bundles it recorded, not their continuing approval (F1). Removal digests read the indexed graph, whose Git and recorded-history inputs are not bound to generation freshness (F2). Retention after drift follows seal order, not replacement lineage (F3). The eighteen milestone seals wait on these three.

### Follow-ups

- [open] F1: make integration approval of a milestone or roadmap follow its descendants: a child's later rejection, lost attestation history or reopening revokes the parent's approval in preparation, verification and closeout (SR-R1-05). Next engine spec, before the eighteen milestone seals. (owner:@oseiaspereira88 crit:high review:2026-11-06)
- [open] F2: bind the Git and recorded-history inputs of indexed removal witnesses to generation freshness, or refuse a witness whose freshness is unknown, for review subjects as for delivery integrity (SR-R1-06, with LR-R1-10 / F4 of pose-delivery-later-removal). Next engine spec, before the eighteen milestone seals. (owner:@oseiaspereira88 crit:medium review:2026-11-06)
- [open] F3: make review retention after drift honour replacement lineage, including backdated or equal seal times, so matching and retained verification choose the same bundle (SR-R1-07). Next engine spec, before the eighteen milestone seals. (owner:@oseiaspereira88 crit:medium review:2026-11-06)
