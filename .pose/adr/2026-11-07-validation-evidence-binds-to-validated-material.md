# ADR: Validation evidence binds to validated material

## Status
Accepted (2026-11-07), by spec `pose-validation-evidence-material-binding`.

## Context

Evidence of a validation run is current for an open spec only while the run's
`scope_provenance[<spec>]` equals `ScopedDeliveryProvenanceDigest`, which
hashes the spec's claims and change sets, including each set's head and
commits. A commit that carries the spec trailer and changes only derived
records (indexes, results, state, assessments, review records, a spec's status)
therefore moves the digest and invalidates evidence that still describes the
same code. Closing B34 reran the 354 s strict matrix a second time for that
reason alone.

## Decision

A validation run records, per scope, a `material_digest`. Per change set it
hashes the head state (path, mode and blob) of every path the set names, both
sides of a rename included, plus the validation matrix. An administrative record
that is still a regular file at the head is left out; a deleted one too. The
scoped digest adds the spec's claims and the engine version. Evidence stays
current while the recomputed digest equals the recorded one, and a present
binding governs: a mismatch is stale and never falls through to the
commit-bound rules.

The administrative classes are exactly: `.pose/indexes/`, `.pose/results/`,
`.pose/state/`, `.pose/assessments/` and `.pose/review-*` directories. They are
owned by the engine and are not configurable by an instance. Everything else,
including every spec file (frontmatter status lines too), `.pose/policy/` and
`.pose/rules/`, is material. The validation matrix file is inside
`.pose/indexes/` but deliberately material. A symlink, submodule or directory
in a class, or a path that left a class, is material. An unreadable tree, an
incomplete checkout or a changed engine makes the evidence stale.

Results without `material_digest` are judged by the existing rule. A scope that
is already `done` keeps its current exemption. Naming the first differing path
would need a stored manifest and is left as a follow-up. The domain returns one verdict
that seal, verify, review-check, closeout-check, close, CLI and MCP all consume.

## Consequences

- A closeout whose later commits are administrative reuses the passing run
  instead of repeating it.
- The result schema gains an additive field, a minor change.
- Adding a class to the list is a contract change that needs its own ADR.

Rejected: caching per-check outputs (stale outputs would pass silently);
configurable class lists per instance (an instance could exempt its own code);
bypassing evidence for any commit that only touches `.pose/` (specs, policy and
rules there change what the checks demand).
