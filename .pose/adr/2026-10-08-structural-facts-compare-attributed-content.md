---
title: Structural facts compare the content the scope's own commits changed
status: accepted
date: 2026-10-08
decision_type: architecture
---

# Structural facts compare the content the scope's own commits changed

## Context

The structural delta compared each subject path between the subject's `base`
and `head`. Those bound every commit between the first and the last attributed
one, so wherever work on several specs interleaved the diff also carried what
other specs and releases did to the same path. The engine already said so in
`range_observations` (`contaminated: its paths are attributed, its base..head is
not`) and still used the range to compare content.

The Harne8 ABM field pilot measured the effect. An independent reviewer
adjudicated 80 of 254 material alerts: 7 false positives (8.75%, Wilson 95%
4.3–17.0%), all from scope mixing in the attributed subject. Recomputing the
universe of 125 bundles showed that the range also decides who owes a mapping:
14 specs whose own change to the validation matrix only appended checks, which
the engine reports and does not charge, were charged because another spec edited
the matrix in between; one spec's own non-additive edit was hidden. The same
work owed a different answer depending on what its neighbours committed.

## Options considered

1. **Mark only**: keep comparing `base..head`, and mark facts on paths changed by
   commits whose trailers also name other specs. Resolves the 3 shared-commit
   false positives; leaves the obligation dependent on interleaving and the facts
   describing other specs' changes.
2. **Compare attributed content and mark shared commits**: record at seal time,
   per path, the blobs at the ends of each uninterrupted run of the scope's own
   commits, compare those, and mark runs that include a commit other specs also
   claim. Chosen.
3. **Also stop exempting check additions**: the reviewer judged the 4 sampled
   check-only additions useful. Rejected for now: the adjudication criterion
   asked whether the reviewer needed to *know or answer* a fact, and a check
   addition is reported, so it is known; it does not show that it needs an
   answer. The exemption was a measured maintainer decision.

## Decision

`ReviewBundleSubject.attribution` lists, per non-renamed subject path, segments
`{before_mode, before, after_mode, after, shared_with}`: blob ids at the ends of
a run of attributed commits whose content chains without interruption, read by
one bounded `git log --no-walk --stdin --raw` over the change sets' commits.
`shared_with` names other specs whose `POSE-Spec:` trailer is on a commit of the
run. The design delta compares each segment instead of `base..head`; a segment
whose ends are equal produces nothing. Facts from a shared run carry
`shared_with`, the plan's material facts carry it, and the plan warns that the
reviewer should map the fact if this scope made the change or answer its mapping
`not-applicable` with the reason. The replay counts `shared_commit_facts`.

## Consequences

- An obligation depends on the scope's own work only, and existing materiality
  rules apply uniformly.
- Measured on the pilot universe: 254 charged facts become 241; 12 are marked
  shared; 14 check-only matrix additions stop being charged; 1 hidden own edit
  becomes material.
- Facts that remain describe the scope's own change, so their digests, and
  display ids, differ from the range reading in newly sealed bundles.
- Mixing inside the scope's own commits, or a declared range that contains a
  release commit, is not detected: the engine does not judge semantics, and the
  reviewer answers such a fact `not-applicable`.

## Invariants

- Blob ids are content identity: `input_digest` still excludes change-set ids and
  provider refs, so a rebase or squash that keeps content keeps the reading.
- The attribution is not implementation identity: the implementation, patch and
  tree digests leave it out; it is sealed with the payload.
- No retroactive governance: bundles sealed without `attribution` are compared
  over their range as before, and so is any subject where Git cannot answer.
- Renames keep the range comparison and their path fact.
- The mark never dismisses a fact; it asks for judgment.
