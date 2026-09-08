---
slug: pose-review-subject-unclassified-removals
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
delivers:
---

# Spec: An unclassified removal does not block a review bundle

## 1. Intent

### Goal
Stop the review subject classifier failing closed on the deletion of a path
whose shape it does not recognise, while keeping it fail-closed on everything
that still exists.

### Business value
A spec that deleted a root-level dotfile cannot be reviewed:

```
[ERROR] unclassified review subject path .harne8-agent-sync.json
```

The file was a local cache that the spec correctly untracked. It is gone, so
there is nothing to read and nothing to classify; the reviewable fact is the
deletion, and the deletion is already in the subject. Refusing the whole bundle
because the shape rules do not recognise a path that no longer exists spends the
entire review to tell the reviewer nothing they cannot already see.

The shape is familiar. `pose-review-subject-submodule-classification` fixed the
same failure for gitlinks in 1.7.11 and
`pose-review-subject-and-scope-precision` fixed the ordering of that fix in
1.8.0: one path the classifier does not know blocks everything. Each time the
remedy has been to teach the classifier the specific case. This is the first
where the honest answer is that the classification does not matter — there is no
content behind the path to review.

### Constraints
- A path that still exists must keep failing closed. Fail-open on an unknown
  file is how a governed subject stops meaning anything.
- The removal stays in the subject. Excluding it would hide a real change to
  make the gate pass, which is the opposite of the intent.

### Non-goals
- Widening the shape rules to recognise arbitrary root-level dotfiles. That
  invites a rule per repository and would not have helped here: the path is gone
  and could not be classified by inspection anyway.

---

## 2. Requirements

### Functional
- R1: A removal the shape rules leave unclassified shall be classified
  `removed`, included in the subject, and shall not block the bundle.
- R2: A created or modified path the shape rules leave unclassified shall keep
  producing `unclassified review subject path` and shall keep blocking.
- R3: A removal shall carry no digest, since there is no content to hash.

### Non-functional
- The existing bundle suite passes unchanged.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — subject classification

### Artifacts
- created: .pose/specs/2026-09-08-pose-review-subject-unclassified-removals.md
- created: .pose/changelogs/unreleased/pose-review-subject-unclassified-removals.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go

### Technical risks
- A path could be removed to get an unknown file past the gate. The removal is
  still recorded in the subject with its own class, so a reviewer sees it; and
  the alternative — refusing to review a deletion — has already cost more than
  the risk it guards.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Classify an unclassified removal instead of blocking (R1, R3)
- [x] Increment 2: Keep creations and modifications failing closed (R2)

### Validation
- [x] Both directions asserted on the same path

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: the third time in one week that an unrecognised path blocked an
  entire bundle. The previous two were fixed by teaching the classifier the
  case; this one has no case to teach.
- Options considered: (a) recognise root-level dotfiles by shape; (b) exclude
  unclassified removals from the subject; (c) classify them as `removed` and
  include them.
- Decision: (c).
- Rationale: (a) would need a rule per repository convention and could not
  classify this path anyway, since the file is gone. (b) makes the gate pass by
  hiding a real change, which is the failure mode this whole subsystem exists to
  prevent. (c) keeps the deletion visible to the reviewer and stops it being
  fatal — the reviewer decides whether deleting it mattered, which is the
  judgement a review is for.

---

## 6. Validation

### Strategy
Assert the same path in both directions, so the change cannot be mistaken for a
general relaxation.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: `.agent-sync-cache.json` is asserted twice — as a removal it must not
  appear in the blockers and must carry class `removed` with no digest; as a
  creation it must still produce `unclassified review subject path`. All eight
  packages pass.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <review_bundle.go — the removal branch precedes the unclassified one and sets class removed with include true; TestReviewBundleDoesNotBlockOnAnUnclassifiedRemoval asserts no blocker names the path>
- R2 [satisfied] <TestReviewBundleStillBlocksOnAnUnclassifiedCreation asserts the same path still blocks when created>
- R3 [satisfied] <the digest branch already skips action removed; the test asserts an empty digest>

### Known gaps
- A rename whose new path is unclassified still blocks, which is correct: the
  new path exists and can be inspected.

---

## 7. Final Report

### Follow-ups

- [open] Consider reporting unclassified paths as a finding on the bundle rather than a blocker, so a reviewer sees them without the gate refusing — owner:unowned crit:low review:2026-12-08
