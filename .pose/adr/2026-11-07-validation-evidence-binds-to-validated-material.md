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

A validation run records, per scope, a `material_digest`. It hashes, in order,
the diffs of the change set's commits that touch at least one path outside a
closed list of administrative classes, plus the claims and the engine version.
Commits whose every path is administrative do not contribute. Evidence stays
current while the recomputed digest equals the recorded one.

The administrative classes are exactly: `.pose/indexes/`, `.pose/results/`,
`.pose/state/`, `.pose/assessments/`, `.pose/review-*` directories, and the
frontmatter `status`/`completed_at` lines of a spec. They are owned by the
engine and are not configurable by an instance. Everything else, including the
body of a spec, `.pose/policy/` and `.pose/rules/`, is material. A commit that
touches any material path is material; a path that is a symlink or arrives by
rename into a class is material. An unreadable path, an incomplete checkout or
a changed engine makes the evidence stale. The validation matrix file is
deliberately outside the administrative classes, so a commit that changes it is
material.

Results without `material_digest` are judged by the existing rule. A scope that
is already `done` keeps its current exemption. The domain returns one verdict
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
