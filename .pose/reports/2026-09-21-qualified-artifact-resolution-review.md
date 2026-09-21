# Qualified artifact resolution — technical review

Date: 2026-09-21. Scope: `spec:pose-qualified-artifact-resolution`.
Reviewer: agent:codex (separate review pass after implementation).

## Rules applied during review

- Change type: Go feature and documentation.
- Workflow: `.pose/workflows/review.md`.
- `.pose/rules/backend-go.md`: checked error paths, bounded traversal and registry locking.
- `.pose/rules/security.md`: checked project policy before external lookup, confinement, malformed references and root-free outputs.
- `.pose/rules/documentation-style.md`: documented one authority and explicit adoption; distinguished readiness from review acceptance.
- No UI, infrastructure or dependency changes; their rules are not applicable.

## Findings and disposition

1. Resolved: portfolio-specific readers excluded dated specs. Projection now uses
   canonical Store enumeration and lookup, with regression across supported layouts.
2. Resolved: unresolved external references could appear non-blocking in projection.
   Unknown/denied/missing targets now block and never select a same-slug local spec.
3. Resolved: project registration alone is insufficient on MCP. The readiness
   request reevaluates and audits the caller's policy for each target project.
4. Resolved: a new policy field alone was unsafe because the old reader is
   permissive. Required adoption uses schema 3/version 1; an actual installed
   5.0.8 binary refused the isolated fixture with `unsupported review policy schema 3`.
5. Resolved: two added direct prints violated the existing CLI rendering guard.
   Error paths now use the shared renderer; the complete suite passed.

## Evidence and limits

- `go -C pose-mcp test ./... -count=1`: passed, including real local Git submodule
  fixtures and HTTP MCP authorization fixtures. These are automated tests, not
  live product smokes or human journeys.
- `pose check --strict`: passed with 11 existing warnings about changelog and old
  capability assessment. No policy or historical evidence was altered to pass.
- `pose assess integrate`: 55 declarations, 54 without a repository consumer
  reference. This scanner inventory does not establish real integration acceptance.
- `pose assess tech-debt`: no debt markers found.
- Source digests describe observed working-tree content; HEAD does not assert
  that this content is committed. mtime is explicitly advisory.
- The module matrix, artifact attribution and current bundle must pass before
  governed closeout. This report alone does not approve a release or federation.

## Decision

The implementation is suitable for the automated closeout gates after source
attribution and matrix execution. Federated evidence acceptance, transfer,
consumer migration and agent routing remain separate scopes. User-directed
human validation remains deferred until after implementation and the 6.x milestone.
