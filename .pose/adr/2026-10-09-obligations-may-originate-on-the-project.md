---
title: Obligations may originate on the project itself
status: accepted
date: 2026-10-09
decision_type: architecture
---

# Obligations may originate on the project itself

## Context

The obligation contract (ADR 2026-10-04, "Obligations are projected; material
action requests are persisted") qualifies every source as
`xref:<project>/(spec|roadmap|milestone):<slug>`. Four producers could not be
projected under it, because what they read belongs to no spec: an open docs
review mark, a capability mechanism with a stale trigger, a release in flight
and a finding recorded in a legacy review record. Attention listed them as
"not projected yet" and reported incomplete coverage for every project that
used one of those sources, so an empty group could never be read as nothing
owed there.

## Options considered

1. **Anchor on a spec where one exists.** A trigger `spec:<slug>` names a
   spec; a manual trigger, a release or a doc owner does not. Coverage would
   stay partial with no way to finish it.
2. **Add a project-level origin.** `xref:<project>/project:<project>` with
   node kinds for the doc and the capability mechanism (release and finding
   already exist). Chosen by the maintainer.
3. **Defer.** Keep the four producers unprojected.

## Decision

`ArtifactRef` accepts kind `project`, whose slug is the project id, and node
kinds gain `doc` and `capability`. The four producers run on every read, so a
phase is never judged by a producer that was not read, and originate on the
project:

- docs-review: each doc with an open mark, owed by the doc's owner when the
  manifest names one, until `pose docs-review resolve`;
- capability-trigger: each non-retired mechanism with stale triggers, until a
  reassessment clears them;
- release: each release newer than the newest verified one that is not
  verified or yanked, except one left prepared or failed while a later
  release was tagged;
- findings: each accepted-risk or follow-up finding past its review date in
  the latest legacy review record of a scope.

All effects are advisory. Free-form investigation notes carry no state and are
not a source.

## Consequences

- Attention reads complete for a project that uses those sources, and shows
  what they owe; on pose-dist the unrecorded publication of v7.0.0 appears.
- The obligation schema stays v1: the change is additive, and existing ids do
  not move because their sources are unchanged.
- A consumer that parsed source refs assuming spec, roadmap or milestone must
  accept `project`.
