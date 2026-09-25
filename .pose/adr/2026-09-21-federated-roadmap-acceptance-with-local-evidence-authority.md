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

## Consequences
Extend existing roadmap/index/review contracts; do not build another gate engine.
Version and explicitly adopt new semantics. Historic bundles keep their sealed
contract. Preview migration without rewriting evidence. A program may remain
blocked after implementations finish; diagnostics must name the unmet criterion
and its owner. Engine acceptance uses generic fixtures, not consumer adoption.
