# ADR: Action effects select material destinations independently of origin

## Status

Accepted — 2026-10-21 for prospective B08 implementation. Complements
[the action-request contract](2026-10-04-obligations-are-projected-action-requests-are-persisted.md).

## Context

The producer currently filters by origin and accepts a subject with a matching
ID of another kind. Roadmap origins bypass the filter. The public effect contract
already defines empty scope as the source artifact, while transports cannot
express explicit scopes. Correcting this can expose previously omitted blockers
in instances that adopted agency readiness; it is a behavioral compatibility
change, not an assertion that every correction is a patch release.

## Decision

Keep origin as provenance, targets as relevance, effect scope as restriction,
and subject as material binding. Select effects through one domain function for
projection and transition gates. Empty scope expands the source artifact; a
roadmap or milestone expands its current members. Explicit scope overrides that
default. Targets alone do not widen blocking or prove independence. Project-wide
queries retain non-superseded requests from terminal origins.

Validate local material references by project, artifact, kind and ID. Resolve a
qualified subject in its own artifact, never by falling back to origin. Use the
existing ArtifactResolver and its authorization boundary for foreign reads;
unresolved references stay errors or explicit observation limits. Do not infer
local restrictions from an unreadable foreign destination. Preserve immutable
journals and request digests; re-evaluate current subjects on reads.

Add structured effect input `{ "phase": "execution", "mode": "block",
"scope": ["xref:project/spec:destination#requirement:R1"] }`. CLI accepts it
through `--effect-json`; MCP accepts strings or objects in `effects`. Existing
`phase:mode` strings keep empty-source scope. Persist the existing qualified
ObligationEffect shape, without changing journal schema or old digests.

Subject binding remains optional in the legacy request contract. A material
approval or acceptance created under adopted agency readiness requires an
explicit subject. Previously persisted unbound requests are not rewritten;
report that their answers provide no content-change detection. Policy-off
creation and transition defaults remain unchanged. This tightening is
prospective and requires compatibility review before any release/adoption.

Rejected alternatives: replacing scope with targets violates empty-source
semantics; filtering by origin loses cross-target restrictions; treating an
unknown roadmap as all specs fabricates global authority; matching only IDs
can bind a decision to a requirement; implicit foreign reads bypass transport
policy; retroactively rewriting journals changes sealed history.

## Consequences

CLI, MCP, query, readiness and governed writes share the destination selection.
Missing membership is visible as limited coverage. New subject refusals affect
only opted-in material approvals/acceptances; no policy is adopted here. Partial
scope names the known restriction without certifying independence of the rest.
B09 authority/waiver, B10 recovery presentation and B11 full snapshots remain
outside this change. Consumed invariant: knowledge:action-answer-proof-and-historical-time.
