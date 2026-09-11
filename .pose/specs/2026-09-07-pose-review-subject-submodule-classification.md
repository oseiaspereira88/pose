---
slug: pose-review-subject-submodule-classification
status: in-progress
created_at: 2026-09-07
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
delivers:
---

# Spec: Review subject classification for submodules

## 1. Intent

### Goal
Let a review bundle seal in a repository that vendors a dependency as a Git
submodule, by classifying the submodule for what it is — a pinned commit —
instead of failing closed on a path it cannot read.

### Business value
`pose review bundle --seal` refuses any repository with a submodule in the
attributed change set:

    [ERROR] unclassified review subject path pose-dist

`reviewBundlePathClass` decides a path's class from the path alone. A submodule
has no shape that distinguishes it from a directory, so it falls through every
branch to the closing `return "", false`, and every seal in that repository is
blocked. Bumping a vendored dependency is one of the most review-worthy changes
a repository makes, and it was the one change that could not be reviewed.

The failure is total rather than partial: a single unclassified path blocks the
whole bundle, so one submodule bump anywhere in a spec's change set makes that
spec unclosable. In the repository that surfaced this, it was the last blocker
standing between a finished spec and its closeout, after every other gate had
been satisfied.

### Constraints
- `reviewBundlePathClass` must stay a pure function of the path. It is called
  from tests with no repository behind it, and giving it filesystem or Git
  access would make classification depend on ambient state.
- A path that cannot be resolved must keep failing closed. Recognising
  submodules must not turn "unknown" into "allowed".

### Non-goals
- Reviewing the submodule's contents. The reviewable unit is the pointer that
  moved, not the tree behind it — that tree has its own repository and its own
  review.

---

## 2. Requirements

### Functional
- R1: A path recorded in the index as a gitlink shall be classified, and shall
  be included in the review subject.
- R2: The subject entry for a submodule shall carry a digest derived from the
  commit it is pinned to, since the path has no file content to hash.
- R3: The entry's reason shall name the pinned commit, so a reader of the
  sealed bundle can see which pointer was reviewed without resolving the
  repository.
- R4: A path that is neither classifiable by shape nor a gitlink shall keep
  producing `unclassified review subject path` and shall keep blocking the
  seal.

### Non-functional
- The existing bundle suite continues to pass unchanged.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — subject construction and digest

### Artifacts
- created: .pose/specs/2026-09-07-pose-review-subject-submodule-classification.md
- renamed: .pose/changelogs/unreleased/pose-review-subject-submodule-classification.md -> .pose/changelogs/v1.7.11/pose-review-subject-submodule-classification.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go

### Technical risks
- Resolution costs one `git ls-files` per otherwise-unclassified path. That set
  is small by construction — it is the set that would have blocked the seal —
  so the call happens on the failure path only, never for a path the shape
  rules already classify.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Resolve gitlinks at the call site, keeping the classifier pure (R1)
- [x] Increment 2: Digest and describe a submodule by its pinned commit (R2, R3)
- [x] Increment 3: Keep unresolvable paths failing closed (R4)

### Validation
- [x] Bundle suite green, new case red without the fix

---

## 5. Decisions

### Decision 1
- Date: 2026-09-07
- Context: where to teach the classifier about submodules.
- Options considered: (a) give `reviewBundlePathClass` the repository root and
  let it ask Git; (b) resolve the gitlink at the call site, which already holds
  `s.Root`.
- Decision: (b).
- Rationale: (a) would make a pure path predicate depend on ambient repository
  state, and its unit tests run against fixtures with no Git metadata. (b)
  keeps the shape rules total and testable and confines the Git question to the
  one place that has a repository to ask.
- Consequences: the resolution runs only for paths the shape rules leave
  unclassified, which is also the cheapest place to put it.

### Decision 2
- Date: 2026-09-07
- Context: what a submodule's digest should be.
- Decision: the SHA-256 of the pinned commit id.
- Rationale: the path is a directory in the working tree and a gitlink in the
  index, so there is no blob to hash. The pinned commit is the entire content
  of the change under review — a bump changes nothing else — and hashing it
  keeps the subject digest sensitive to exactly what moved.
- Consequences: `submodule` joins the class vocabulary. It is deliberately not
  folded into `governance`, whose entries are carried in the non-sensitive
  criterion contract; a dependency pointer belongs with implementation.

---

## 6. Validation

### Strategy
Assert both directions of the gate: the new case must pass with the fix and
fail without it, and the pre-existing rejection of a genuinely unknown path
must survive.

### Deterministic checks

#### Test
- Command: `go test ./... -count=1`
- Scope: `pose-mcp`
- Expected: exit 0

#### Regression
- Command: `go test ./internal/pose/ -run TestReviewBundleClassifiesSubmodulePath`
- Scope: the new case, with the gitlink resolution disabled
- Expected: fails with `submodule was treated as unclassified`

### Execution log
- Date: 2026-09-07
- Environment: local, Go 1.26
- Notes: the fixture builds a gitlink with
  `git update-index --add --cacheinfo 160000,<sha>,vendor/dep`, so it needs no
  network and no checked-out submodule. Disabling the resolution reproduced the
  original blocker verbatim: `unclassified review subject path vendor/dep`. All
  eight packages pass with the fix.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <review_bundle.go — reviewBundleGitlinkSHA resolves mode 160000 and the entry is included; TestReviewBundleClassifiesSubmodulePath asserts class submodule>
- R2 [satisfied] <review_bundle.go — the digest branch hashes the pinned commit instead of calling reviewBundleFileDigest on a directory; the test asserts a non-empty digest>
- R3 [satisfied] <entry.Reason carries "attributed submodule pinned to <sha>"; asserted by the test>
- R4 [satisfied] <TestReviewBundleRejectsUnclassifiedSubjectPath still passes: mystery.data is not a gitlink and still blocks the seal>

### Known gaps
- A submodule whose pointer is unchanged but whose contents drift in the
  working tree is reported by `reviewBundleWorkingTreeChange` as dirty, which is
  the correct signal but names the directory rather than what changed inside it.
- This spec ships in v1.7.11 without being closed, which departs from the
  convention of the previous release commits. Closing it requires an
  attestation, and `pose review attest` fills any criterion whose evidence
  class has no producer with `<class>:auto-attest` — the fallback at
  `review_bundle.go`. `spec-closeout@1` asks for `validation` and `test`, and
  `ValidEvidenceClasses` (`delivery_surface.go`) accepts neither, so those
  criteria cannot be satisfied by a real check on this engine. The existing
  attestation carried into v1.7.10 (`rva-2e2da688435296b9`) is auto-attest for
  exactly those criteria. An attestation that records no judgment is worse than
  an open spec: it makes the gate report green while proving nothing. The
  release gates do not require terminal scope for this cut and all pass, so the
  spec stays honestly open until the two evidence vocabularies are reconciled.

---

## 7. Final Report

### Follow-ups

- [open] Report the submodule's own dirty detail rather than just the directory (owner:unowned crit:low review:2026-12-07)
