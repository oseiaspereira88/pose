# ADR: Explicit remediation links in spec frontmatter

## Status
Proposed — implemented incrementally by `pose-abm-remediation-lineage`.

## Context
Governance outcomes deliberately cannot infer remediation from commit order,
similar titles, `depends_on`, or a reviewer's identity. A finding ID is local to
an attestation; referring to `F1` alone is ambiguous. POSE frontmatter is flat.
The existing append-only projection ADR leaves lineage to an explicit contract.

## Decision
Use an optional comma-separated `remediates` list of `reference@category` values.
References are `spec:<slug>` or `finding:<attestation-id>/<finding-id>`. Allow only
`defect-fix`, `simplification`, `revert`, `requirement-change`, `planned-evolution`.
Load the existing immutable attestation and bundle to resolve a finding's scope;
do not rewrite either. A finding on a spec links to that spec in cycle detection.
Reject orphan, duplicate, ambiguous, cyclic, escaping or oversized graphs in
lint/DoR and bundle preparation. Bound traversal at 256 specs and 1024 links;
reject symlinks in the spec registry when lineage validation is requested.

Include nonempty links, sorted, in the semantic review subject and legacy scope
digest. No field means no new semantic bytes or traversal. Registration is an
explicit spec edit, not an inferred relation or a second event journal.

## Consequences
CLI and MCP spec readers expose the same optional list. Existing sealed records
without lineage remain byte-compatible; readers predating this contract must be
upgraded before consuming bundles that opt in. No profile is activated.
The first increment establishes lineage integrity only: a valid declaration is
not proof of causal truth, successful remediation or approval. Maturity, censoring
and category-specific rates require a later projection increment in the same spec.

Rejected alternatives: a free-text field loses validation; a new sidecar journal
duplicates authority; title matching manufactures causality; finding ID alone
cannot identify an immutable observation. Accepted risk remains a finding
disposition, not another spelling of defect correction.
