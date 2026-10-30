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
- R4: The roadmaps `delegated-review` and `delegation` shall seal their milestone bundles with these rules.

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
- created: .pose/specs/2026-10-30-pose-review-subject-later-removal.amendments.jsonl
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-subject-later-removal module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_closeout.go

### API/contract changes

A review subject entry may carry a removal digest; a milestone bundle's child may be a retained closed review.

## 4. Tasks

- [ ] Removal witness in the review subject
- [ ] Retained children in milestone seals
- [ ] Seal the eighteen milestone bundles
- [ ] Independent review and closeout

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

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
