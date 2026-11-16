# ADR: Federated roadmap acceptance with local evidence authority

## Status
Accepted — 2026-09-24. The user authorized the documented implementation
sequence. Implemented by
[federated acceptance](../specs/2026-09-21-pose-federated-roadmap-acceptance.md),
using the [identity contract](2026-09-21-qualified-artifact-authority-and-explicit-spec-transfer.md).

## Context
Roadmap membership and closeout resolve local specs. Portfolio status/mtime is
advisory, not delivery evidence. A mirrored coordinator spec demands a second
implementation review; accepting remote done alone bypasses composition.

## Decision
- Compose programs with qualified spec/milestone/roadmap references. A local
  roadmap becomes a sub-roadmap through an explicit outcome-consumption edge,
  never through filesystem nesting. Create one only for a cohesive local gate.
- Distinguish ownership, prerequisite and outcome-consumption edges. Keep at most
  one owning active roadmap per executable spec identity; consuming that roadmap
  does not own its specs again. Direct external spec membership remains possible
  if no other active roadmap owns it. Deduplicate progress by qualified identity.
- Detect cycles across the authorized closure, including mixed edges and
  redirects. Unknown closure does not prove acyclicity. Apply traversal limits.
- Seal an external dependency manifest with identities, pinned source revisions,
  artifact/subject/plan digests, governing contracts, policy digests and verifier
  results/evidence. Build one bounded snapshot, not repeated live-HEAD reads.
- For same-project coordinator and dependencies, record the last commit that
  changed the artifact file as its source revision. Verify the current file
  against committed content and retain a repository-HEAD snapshot to detect
  concurrent changes during resolution. Keep the full repository revision as
  the pin for external projects, including gitlinks and adopted trust.
- Verify children under source authority plus explicitly adopted consumer trust
  requirements. Recheck relevant pinned inputs immediately before application;
  changes stale the coordinator review. Scope caches by digests and authorization.
- Treat a passing source check as delivery proof when its sealed observation is
  `observed`, or `carried-forward` with a recorded Git head after the reviewed
  subject and no later than the pinned source revision. Require a fresh,
  approved source bundle and its current provenance in either case. A check
  before the subject, with no commit, or outside the pin cannot prove delivery.
- A spec whose `depends_on` names another project's artifact is a coordinator
  for those edges only: its bundle seals their manifest and its closeout
  reports their blockers. Local dependencies keep lifecycle rules, and a spec
  without an external edge keeps its previous payload. Missing roots block
  closeout rather than failing the read (2026-09-26,
  [spec-scope acceptance](../specs/2026-09-26-pose-federated-spec-dependency-acceptance.md)).
- A milestone is a coordinator for the roadmap's own prerequisites and
  consumed outcomes plus its own predecessors, members and consumed outcomes;
  members of other milestones do not gate it. Milestone bundles sealed with
  the roadmap-wide manifest keep that sealed meaning and stay fresh while it
  is unchanged (2026-09-26,
  [milestone-scoped acceptance](../specs/2026-09-26-pose-federated-milestone-scoped-acceptance.md)).
- The roadmap gate judges a member owned by another project through federated
  acceptance, not local closeout, and `pose close` of a milestone gates on
  that milestone's acceptance and members without the roadmap cut criteria
  (2026-09-26,
  [roadmap gate scopes](../specs/2026-09-26-pose-roadmap-gate-scopes-milestones-and-external-members.md)).
- Separate implementation closure, consumer adoption and program acceptance.
  Keep composition criteria and explicit outcome review at the coordinator.
  Required remote done is insufficient without valid evidence and composition.
- Distinguish unknown, unavailable, unauthorized, stale, conflicting and
  unsupported-contract states. Unresolved required edges block acceptance.
  Mtime remains advisory; restricted metadata and filesystem roots stay hidden.
- A submodule adoption claim must match the consumer gitlink. A program tracking
  another revision must label that distinctly. Sibling checkouts obey the same
  identity rules. Never fetch, write another project or trigger its review as a
  side effect of reading/checking a coordinator.

Rejected: copying/relabeling bundles, auto-closing parents from child statuses,
requiring a transaction across all repos, mandatory sub-roadmap per repository,
and a hosted authority replacing local governance. Also rejected: using local
repository HEAD as the manifest identity for an unchanged artifact; evidence
commits would stale its review without changing the governed contract.
Rejected: requiring exact subject and validation commits to be equal. It
would reject validation recorded after a reviewed implementation without
adding a stronger content guarantee.

## Consequences
Extend existing roadmap/index/review contracts; do not build another gate engine.
Version and explicitly adopt new semantics. Historic bundles keep their sealed
contract. Preview migration without rewriting evidence. A program may remain
blocked after implementations finish; diagnostics must name the unmet criterion
and its owner. Engine acceptance uses generic fixtures, not consumer adoption.

## Amendment — 2026-11-16: a closed scope stands over a moved trust pin

Spec `pose-closed-scope-stands-over-a-moved-federated-pin`. A dependency of
another project is accepted at the whole-repository revision the consumer's
trust pin names. Spec `pose-closeout-judges-the-manifest` (2026-11-01) made
closeout agree with `review verify`: any difference between a closed scope's
sealed manifest and the current one unapproves it. So every move of a pin
unapproved every closed scope that depends on that project. After harne8
pinned pose-dist at `edcf5f2`, 18 of its closed specs lost their approval.
The three dependencies of one of them had only had follow-up dispositions
recorded and been reviewed again.

A closed scope's approval now stands over such a dependency when it is still
the commitment the reviewer accepted:

- the same reference, identity and relationships;
- resolved and done, then and now;
- the same plan digest and governing contracts;
- its spec, read in its own project at the sealed and the current revision, is the same before its Final Report (intent, requirements, plan, decisions, validation), found in its CommonMark structure, or, for a section-split spec, in every section but `final-report.md`: a plan digest covers the review procedure, not the requirements;
- a current review bundle;
- no blocker of the current manifest naming it.

The current acceptance already requires current trust and an approved
review of the dependency. Its revision, content, evidence, implementation
digest and review may move with the pin. The trust policy digest, which holds
the pins, follows only when at least one dependency of another project moved
and was judged this way; a trust policy change alone still unapproves. A
same-project dependency keeps the rule it had.
