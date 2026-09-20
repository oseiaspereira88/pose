---
slug: pose-dist-records-its-own-contract-adoption
status: in-progress
created_at: 2026-09-20
completed_at:
supersedes:
depends_on: pose-abm-progressive-review
priority: 1
components:
task_type: refactor
delivers:
---

# Spec: this repository records the contract adoption it already practises

## 1. Intent

### Goal

Record in this repository's own policy the dated adoption of `explicit-judgment`
and `structural-causality`, which its reviews have been sealed under since the ABM
integrity work but which its policy never declared.

### Business value

The `pose-abm-foundation` roadmap's bootstrap invariant asks both repositories to
readopt the corrected judgment contract. Measured on 2026-09-20: 34 of this
repository's 521 sealed bundles already list `explicit-judgment`, so the substantive
half has been true here for several specs — but the policy recorded no adoption at
all, while the Harne8 instance recorded both. The invariant was satisfied asymmetrically:
one side by practice with no declaration, the other by declaration.

A declaration that is absent is not the same as a practice that is present. Anyone
reading this policy to learn which contracts govern the repository would have
concluded none of the two did, and the dated exemption that protects work completed
before a contract arrives had nothing to stand on here.

### Constraints

Do not activate any overlay profile. Completed closeouts keep their approval. The
policy must stay readable by the previously released engine. The engine's own
adoption path must be what writes it, not a hand edit.

### Non-goals

Changing which bundles are governed: that is decided per bundle by the sealing
engine, not by this date, so the stamp adds no enforcement. Activating
`engineering-judgment`, `high-criticality-review` or `structural-materiality`.

## 2. Requirements

- R1: The policy records both contracts, dated, written by the engine's own
  adoption path.
- R2: Nothing else in the policy changes semantically.
- R3: No overlay profile is activated and completed closeouts keep their approval.
- R4: The policy stays readable by the previously released engine.

## 3. Technical Plan

`pose update --no-self` from the candidate engine. In this repository the embedded
distribution carries neutral policy templates rather than the live policy, and
config seeding only writes absent files, so the live policy is not replaced — only
`stampContractAdoption` touches it. The managed manual is skipped here for the same
self-referential reason, which the run reports as
`skipped POSE.md: unresolved scaffold placeholder`.

### Artifacts

- created: .pose/specs/2026-09-20-pose-dist-records-its-own-contract-adoption.md
- created: .pose/changelogs/unreleased/pose-dist-records-its-own-contract-adoption.md
- modified: .pose/policy/review.json
- modified: .pose/state/machinery-manifest.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/spec-graph.json

### Rollout and reversal

Both paths are tracked; reverting the stamp is reverting the commit. Removing the
dates would not remove the contracts from any sealed bundle.

## 4. Tasks

- [x] Run the engine's adoption path and confirm the scope of what it wrote.
- [x] Prove the policy changed only by the two additions.
- [x] Prove overlays are untouched and completed closeouts keep their approval.
- [x] Prove the previously released engine still reads the policy.

## 5. Decisions

The stamp is recorded as its own spec rather than folded into
`pose-abm-progressive-review`, which is closed and whose sealed review is bound to
its body: amending it would have staled a valid closeout to carry a change that was
never in its scope.

The write reorders the whole policy document alphabetically, because the engine
rewrites it through `json.MarshalIndent` to preserve keys it does not model. That is
accepted rather than worked around: a hand edit that kept the original ordering
would not be the engine's adoption path, and the reordering is proven semantically
empty below.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Adoption recorded | `pose update --no-self` then read `.pose/policy/review.json` | Both contracts dated 2026-09-20 |
| Nothing else changed | parse the policy before and after and diff the key sets | Only the two keys added; none removed or changed |
| Overlays and closeouts intact | `pose closeout-check spec:<done-slug>` | `review_approved=true`, `terminal=true` |
| Previous engine reads it | `pose review-plan <scope>` with the installed 5.0.8 binary | Exit 0 with a resolved plan |

### Execution log

2026-09-20 UTC, candidate engine built from this repository. `pose update --no-self`
reported the machinery merges as no-ops, skipped `POSE.md` as an unresolved scaffold
placeholder — the self-referential protection working — and stamped
`explicit-judgment=2026-09-20` and `structural-causality=2026-09-20`.

Measured rather than assumed:

- Parsing the policy before and after: two keys added, none removed, none changed.
  The 30-line diff is `json.MarshalIndent` reordering keys alphabetically.
- `overlay_profiles` unchanged at `backend-review@1`, `frontend-review@1`.
- `pose-abm-progressive-review`, `pose-abm-structural-delta` and
  `pose-abm-review-soundness` all report `review_approved=true` and
  `terminal=true` after the stamp.
- The installed 5.0.8 binary resolves a review plan against the reordered policy,
  exit 0. It still refuses bundles carrying `implementation_digest`, which is the
  bundle schema and predates this change: 34 such bundles existed here before it.

### Requirement trace

- R1 [satisfied] `.pose/policy/review.json` carries `explicit_judgment_adopted_at`
  and `structural_causality_adopted_at` dated 2026-09-20, written by
  `stampContractAdoption` during `pose update --no-self`
- R2 [satisfied] before/after key-set comparison: two added, zero removed, zero changed
- R3 [satisfied] `overlay_profiles` unchanged; three completed specs keep
  `review_approved=true` and `terminal=true`
- R4 [satisfied] the installed 5.0.8 binary resolves a review plan against the
  readopted policy, exit 0. Scoped to the policy deliberately: sealed bundles are a
  separate artifact and a separate, pre-existing incompatibility

## 7. Final Report

### Scope delivered

This repository now declares the two contracts its reviews were already sealed
under. Both repositories carry the dated adoption, so the roadmap's bootstrap
invariant is satisfied symmetrically instead of half by practice and half by
declaration.

### Residual risks

The stamp adds no enforcement: whether a bundle is governed is decided by the engine
that sealed it. A pinned 5.0.8 still cannot read bundles carrying
`implementation_digest`, which is unchanged by this spec and recorded in
`harne8-readopts-explicit-judgment`.

### Follow-ups

None. Activating the ABM overlay profiles remains an explicit adoption decision.
