---
slug: review-policy-adoption-is-the-instances
status: in-progress
created_at: 2026-09-20
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: review adoption dates belong to the instance that earned them

## 1. Intent

### Goal

`.pose/policy/review.json` is not in `SelfReferentialPolicyFiles`, so this
repository's live review policy is copied byte-for-byte into the embedded
distribution. Every `pose install` therefore hands a target project this
repository's review adoption dates instead of stamping the target's own.

### Business value

This is the defect class `pose-changelog-adoption-is-the-instances` closed for
`changelog.json`, still open for the larger policy beside it. That spec's own words
apply unchanged: a date belonging to someone else "silently exempts everything
completed before a date belonging to someone else".

Measured on 2026-09-20, the embedded template already ships `adopted_at: 2026-08-02`,
`component_aware_adopted_at: 2026-08-13`, `review_bundles_adopted_at: 2026-08-14`
and `evidence_vocabulary_reconciled_at: 2026-09-08`. Recording this repository's own
readoption added `explicit_judgment_adopted_at` and `structural_causality_adopted_at`
to the same channel, which is how the leak was found: the parity test refused the
drift, and the only ways to clear it were to ship the dates or to give up recording
them. Neither is acceptable, and the second is why this cannot be worked around in
the repository that owns the file.

`stampContractAdoption` skips any contract already recorded, so a fresh instance that
receives these dates is never stamped with its own. The exemption it inherits is not
even consistently conservative: for a project installing later than a leaked date,
the inherited date is stricter than it deserves; for one migrating in with specs
completed between its own start and a leaked date, it is laxer — work nobody reviewed
under the contract is exempted from it.

### Constraints

The neutral template a fresh instance receives decides what review means in every new
repository, so it is a contract decision and not a mechanical extraction. It has to
stay a valid schema-v2 policy that `pose install` can operate, and it must not silently
turn review off or on for existing instances, whose live file is never replaced by
seeding.

### Non-goals

Changing what this repository's own policy says. Changing which bundles are governed:
that is decided per bundle by the sealing engine.

## 2. Requirements

- R1: `review.json` is shipped as an explicit neutral template rather than as a copy
  of this repository's live policy.
- R2: A fresh `pose install` receives no adoption date it did not earn, and
  `stampContractAdoption` then stamps the day that instance received each contract.
- R3: An existing instance's live policy is not modified by the change, and its
  recorded dates keep their meaning.
- R4: The neutral template is a valid schema-v2 policy the engine can operate from,
  pinned by a test that installs from it and resolves a review plan.
- R5: The parity contract still refuses drift, with `review.json` compared against its
  neutral template rather than against live content.

## 3. Technical Plan

Add `review.json` to `distpolicy.SelfReferentialPolicyFiles` and write its neutral
content in `NeutralPolicyTemplates`, the way `changelog.json` is handled. Decide the
template's contents deliberately — which scope profiles, whether `component_aware`
and `review_bundles` start enabled, and no dates at all — and pin them.

### Artifacts

- created: .pose/specs/2026-09-20-review-policy-adoption-is-the-instances.md
- created: .pose/changelogs/unreleased/review-policy-adoption-is-the-instances.md
- created: pose-mcp/internal/cli/review_policy_adoption_test.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/review.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/releases.json
- modified: .pose/indexes/spec-graph.json

### Rollout and reversal

The change affects fresh installs only, because seeding never replaces an existing
policy. Reverting restores the copy, which is why the test in R5 is what holds the
contract rather than a convention.

## 4. Tasks

- [x] Pin the current leak in a test: install into a fresh target and assert it
      receives no date it did not earn.
- [x] Decide and write the neutral template's contents.
- [x] Ship it through the self-referential exclusion and prove an existing instance
      is untouched, including under `--force`.

## 5. Decisions

Recorded as its own spec, at priority 0, rather than fixed inside
`pose-dist-records-its-own-contract-adoption`: choosing what review means in every
new repository is a contract decision that deserves its own review, and folding it
into a stamp would have hidden it behind a two-line policy change.

The stamp that exposed it was kept rather than reverted. Reverting would not have
fixed the leak — four dates were already travelling — and it would have removed the
one honest record of adoption in the repository that practises it. The cost is two
more fields on a channel that already leaks, stated in that spec's residual risks.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Fresh install earns its own dates | `go test ./internal/cli -run InstallStampsItsOwnReviewAdoption -count=1` | No inherited date; each contract stamped with the install day |
| Existing instance untouched | `go test ./internal/cli -run UpdateKeepsRecordedAdoption -count=1` | Live policy and its dates unchanged |
| Template is operable | `go test ./internal/cli -run NeutralReviewPolicyResolvesAPlan -count=1` | A plan resolves from the neutral template |
| Parity holds | `go test ./internal/scaffold -run TestEmbeddedDistMatchesPoseDist -count=1` | `review.json` compared against its neutral template |

### Execution log

2026-09-20: found while recording this repository's own contract adoption. The parity
test failed with `differs: [.pose/policy/review.json]`, which is what revealed that
the file is shipped as a byte copy. The embedded template's leaked dates were read
directly and are listed above.

2026-09-20, implemented. The leak was pinned first: a fresh `pose install` was
asserted to receive no date it did not earn, and the test reproduced the defect
exactly — `adopted_at=2026-08-02`, `component_aware_adopted_at=2026-08-13`,
`review_bundles_adopted_at=2026-08-14`, `evidence_vocabulary_reconciled_at=2026-09-08`,
plus the two recorded that day, plus this repository's adopted
`overlay_profiles`. The overlays were the part the original finding had not
measured: every fresh install was adopting the overlays *this* repository chose,
which is the decision explicit adoption exists to keep with the project.

Two things the implementation had to decide rather than extract:

- The template ships the current contract shape — review enabled, component-aware
  and review-bundles on, the three shipped scope profiles — with no dates and no
  overlays. Keeping the shape preserves what a fresh install gets today; dropping
  the dates and overlays removes what belongs to this repository.
- `adopted_at` is the policy's own, not a registry contract, so nothing stamped it.
  Shipping it empty would have gated an instance's entire history, and shipping this
  repository's value would have exempted a slice of someone else's. It is now
  stamped like the contracts, with the same rule: absent means stamp, explicitly
  empty stays a decision.

Being dateless makes the template unloadable on its own — enabling component-aware
review requires the date that says when the instance received it. That coupling is
asserted rather than hidden, because the template is an input to seeding, which
seeds and stamps in one step.

Defect injection, and one correction to my own method: removing `review.json` from
the exclusion list did **not** reproduce the leak, because the generator writes the
neutral template after the sync — the template entry, not the exclusion, decides the
shipped bytes. Removing the template entry did reproduce it, and the install test
caught all six dates. Removing the `adopted_at` stamp failed five tests including
the three brownfield kits. The first injection was measuring the wrong half of the
mechanism, which is exactly the failure this repository keeps re-learning.

### Requirement trace

- R1 [satisfied] `TestSelfReferentialPolicyFilesExcluded`,
  `TestEmbeddedDistMatchesPoseDist` — `review.json` is excluded from the wholesale
  sync and compared against its neutral template
- R2 [satisfied] `TestInstallStampsItsOwnReviewAdoption` — a fresh install carries
  no date it did not earn, and every registry contract carries the install day
- R3 [satisfied] `TestUpdateKeepsRecordedAdoption` — a recorded adoption survives
  `update --no-self` and `install --force`, and so do the instance's own overlays
- R4 [satisfied] `TestNeutralPolicyTemplatesAreSchemaValidAndInert` — the template
  is dateless, refuses to load undated for the stated reason, and loads with the
  contract shape once stamped; `TestInstallStampsItsOwnReviewAdoption` resolves a
  review plan from the installed result
- R5 [satisfied] `TestSelfReferentialPolicyFilesExcluded`,
  `TestNeutralPolicyTemplatesAreSchemaValidAndInert`

## 7. Final Report

### Scope delivered

A distribution no longer hands a project another repository's review history. The
shipped template carries the contract shape and nothing dated; `pose install` stamps
the day the instance received the policy and each governed contract; an existing
instance keeps what it recorded, including under `--force`.

### Residual risks

The template's contents are now a contract: changing which profiles or flags a fresh
instance receives changes what review means in every new repository, and the tests
pin the current answer rather than justify it forever. `migrateInstanceReviewPolicy`
still carries hardcoded `2026-08-13`/`2026-08-14` fallbacks from this repository's own
history for schema-v1 instances that have no `adopted_at` to derive from — the same
leak class, in the migration path rather than the distribution, and untouched here.

### Follow-ups

- The v1 migration fallback dates above (owner: @pose-maintainers crit: medium review: 2026-10-20)
