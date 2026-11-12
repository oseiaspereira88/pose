# ADR: Closeout review policy by cost and benefit

## Status
Accepted — the maintainer approved this base explicitly on 2026-11-12, including
legacy compatibility and `always` requiring a one-member milestone's own review.
This records the architectural decision, not policy adoption or a delivery attestation.

## Context

Spec [pose-closeout-review-policy-by-cost-benefit](../specs/2026-11-11-pose-closeout-review-policy-by-cost-benefit.md)
separates the mandatory close of an aggregate scope from its independent review.
The current [one-member milestone rule](2026-11-02-a-milestone-of-one-member-is-that-member.md)
inherits the member's approval, while multi-member milestones and roadmaps normally
require their own reviews. Review value depends on composition; the maintainer
chose explicit deterministic thresholds instead of a judgment made by an agent.

Sealed bundles include policy and review plans. A new setting must not reinterpret
the historical contract, erase a rejection, skip member reviews, or infer adoption
from an unrecognized optional field.

## Decision

Store `milestone_review_required` and `roadmap_review_required` in the versioned
instance review policy `.pose/policy/review.json`. Accept exactly `never`,
`when-justified`, and `always`; reject unknown values before preparing a current
plan or writing a closeout. Use one domain evaluator across CLI, MCP, planning,
verification, state, closeout and federation; expose its reason in their projections.

- `never`: require no independent review of that aggregate scope.
- `always`: require the scope's own independent review, including a one-member
  milestone. An inherited member approval does not satisfy this explicit choice.
- `when-justified`, milestone: require review exactly when it has more than two
  distinct canonical spec members. Duplicate aliases do not increase the count.
- `when-justified`, roadmap: require review exactly when its union of milestones
  contains more than one distinct canonical spec and at least one resolved ADR
  association. Count the spec once even if it appears in multiple milestones.

Recognize an ADR association declared by the roadmap or a canonical association
back from an ADR. Resolve explicit references, not incidental words in prose.
Unknown, unreadable or ambiguous member/association inputs cannot yield an
exemption: report the missing knowledge and refuse the close until resolved.
Acceptance tests must fix the reference syntax in the implementation spec before
changing its parser.

The defaults for newly installed/adopting instances are `never` for both keys.
Introduce a review-policy schema version that older engines reject. A legacy
policy with neither key retains its legacy semantics until explicit adoption;
setup/settings show the effective legacy rule and the proposed new values before
writing both keys. Do not silently inject `always` (which differs for one-member
milestones), and do not silently disable review on an existing instance by parsing
an absent field as `never`. A partially configured adopting policy is invalid.

After adoption, `pose close` remains an explicit operation for every milestone
and roadmap in all three modes, including scopes whose review is inherited or
dispensed with. Persist a closeout receipt containing `review: not-required`,
the exact policy key/value, evaluator version and reason/counts when exempt.
Satisfying member reviews or reaching the last member does not itself perform
this close. Member reviews and edge/exit-gate validation remain required.

Optional reviews use the same verification as required reviews. An authentic
negative review retains its veto until remediated under the existing rules;
changing a setting cannot erase it. No exemption may weaken a stronger protected
baseline or mandatory-human requirement. `doctor` reports open scopes newly owing
review after policy or composition changes; the close gate enforces the result.

Verify scopes already sealed as done against their sealed policy and plan,
including legacy inherited milestones. Never backfill a fictitious exemption,
reinterpret an existing receipt, or invalidate historical approval merely because
the current setting differs. Apply the new close and exemption rules prospectively.

## Consequences

New instances skip aggregate reviews by default while still closing each scope
and recording why review was not required. `when-justified` preserves review for
the compositions the maintainer selected. Explicit `always` adds cost even for
a one-member milestone; that is deliberate and visible.

Implementation touches review-policy loading, common scope evaluation, persisted
closeout evidence, setup/settings, doctor, CLI/MCP projections and their tests.
The existing one-member ADR is superseded only for newly adopting open scopes;
its sealed historical outcomes remain valid. No automatic instance adoption,
human approval or delivery attestation is implied by this proposed ADR.

Rejected alternatives:

- Boolean switches: cannot express the requested middle mode.
- An agent estimates the benefit: yields irreproducible gates.
- Require review with two milestone members: contradicts the chosen threshold.
- Default existing missing fields directly to `never`: silently changes authority.
- Treat `always` as legacy behavior: hides the one-member exception.
- Ignore a recorded rejection after switching to `never`: launders negative evidence.
- Recompute done scopes using current policy: breaks historical sealed approvals.
- Remove close when review is exempt: conflates lifecycle and review requirements.
