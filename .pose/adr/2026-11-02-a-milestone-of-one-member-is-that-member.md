# ADR: A milestone of one member is that member

## Status
Accepted (2026-11-02), spec `pose-milestone-review-proportional`.

## Context

100 of the 133 milestones with members have exactly one. Reviewing such a milestone repeats the review of its spec: it took 42 milestones and a night to learn that. Of the 34 judged, the two defects worth having came from an adversarial reader with a new falsifier, not from the milestone: the gate of `manifest-tag-integrity` restates R4 of its spec, and its registered check passed with the defect in place. A declared exit check would have passed too.

## Decision

- A milestone with exactly one local member has no review of its own: its approval is the approval of that spec, once the spec is closed.
- A negative review recorded on the milestone stands, and an approval already recorded on it stays as it is.
- While that spec is open, its review answers for the exit gate of the milestone, as a criterion of the plan, so the gate is judged once and not never.
- A milestone of several members keeps its integration review, where the review of `community-launch` found what no spec could.
- A roadmap may declare `unreviewed_milestones: <reason>`, and its reached milestones then stand without a review. The reason is sealed with the roadmap, whose own review stays required.

## Consequences

- One rule answers `review verify`, closeout, the review of a roadmap and the federated proof, because all of them read one verification.
- No bundle or attestation exists for an inherited milestone. `review verify` shows the bundle of the spec and says it is inherited.
- The second look that a late milestone review happened to give is gone for one-member milestones, and belongs to the review of the roadmap.

## Alternatives

- Inheriting only with a declared `exit-check`: a check the author registers passes by construction, and it would have passed over the defect that motivated it.
- Keeping every review: a repetition that costs a bundle of thousands of lines per milestone.
- Dropping milestone review altogether: it is where several specs meet, and the only place that sees that.
