---
slug: pose-review-subject-and-scope-precision
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
delivers:
---

# Spec: Precision in the review subject and its component scope

## 1. Intent

### Goal
Correct two defects in work shipped days earlier, both found by review of the
pull requests that shipped them: a submodule is only recognised when no other
rule claims its path, and a component-scoped `validate` tool accepts evidence
from a component it does not govern.

### Business value
Both are narrow, both are in code released as 1.7.11 and 1.7.12, and both were
introduced by fixes whose own tests exercised the case that works.

**A gitlink under a classified path is unsealable.** `pose-review-subject-submodule-classification`
resolved the gitlink only when the path-shape rules left the path unclassified:

```go
if class == "" {
    if sha, ok := reviewBundleGitlinkSHA(s.Root, path); ok { ... }
}
```

A submodule under a mapped component, or under a recognised prefix such as
`extensions/`, `scripts/` or `src/`, is classified as `implementation` first, so
the lookup never runs. The digest step then calls `reviewBundleFileDigest` on a
directory and the bundle fails with `cannot be read`. The original test used
`vendor/dep`, which matches no prefix and no component — it exercised the one
layout where the guard happens to fire. The repository that motivated the fix
vendors its dependency at the root, so the gap never surfaced there either.

**Overlay evidence classes leak across components.** `pose-review-plan-producible-evidence-classes`
replaced a hardcoded class with the union of the classes every selected profile
declares for `validate`, and assigned that union to every component-scoped tool.
Review verification accepts any one listed class, so on a scope with a Go API
overlay demanding `unit` and a web overlay demanding `e2e`, the API's
disposition can be satisfied with the web app's evidence and the reverse. That
is the opposite of component-level provenance, which is the entire point of a
component-aware plan.

Neither is a regression against the state before those fixes — the first case
was previously unsealable for a different reason, and the second demanded a
class no check could emit at all. Both are the difference between a fix that
works in the layout that motivated it and one that works generally.

### Constraints
- `reviewBundlePathClass` stays a pure function of the path. Being a gitlink is
  a fact about the index, and threading Git access into a predicate that is unit
  tested without a repository would trade one defect for a worse one.
- The gitlink lookup must stay bounded. Asking Git once per subject path would
  turn a bundle over a large change set into hundreds of process launches.

### Non-goals
- Reviewing a submodule's contents. The reviewable unit remains the pointer that
  moved; the tree behind it has its own repository and its own review.

---

## 2. Requirements

### Functional
- R1: A path recorded in the index as a gitlink shall be classified as a
  submodule regardless of what the path-shape rules would otherwise return.
- R2: Gitlink resolution shall cost one Git invocation per bundle, bounded by
  the subject's own paths, not one per path.
- R3: A component-scoped tool shall carry only the evidence classes declared by
  profiles that govern that component — the base profile, and the overlays whose
  selectors matched it.
- R4: The repository-wide tool shall keep the union across all selected
  profiles, since it answers for every component.

### Non-functional
- The existing bundle and plan suites pass unchanged.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go` — gitlink resolution and precedence
- `pose-mcp/internal/pose/review_plan.go` — per-component evidence scoping

### Artifacts
- created: .pose/specs/2026-09-08-pose-review-subject-and-scope-precision.md
- renamed: .pose/changelogs/unreleased/pose-review-subject-and-scope-precision.md -> .pose/changelogs/v1.8.0/pose-review-subject-and-scope-precision.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_plan_test.go

### Technical risks
- Scoping narrows what a component-scoped tool accepts. A project relying on the
  union to satisfy one component with another's evidence would start failing —
  which is the defect being corrected, not a cost to preserve.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Resolve every gitlink in one call before classification (R1, R2)
- [x] Increment 2: Let the gitlink take precedence over the path-shape class (R1)
- [x] Increment 3: Scope overlay evidence to the components it matched (R3, R4)

### Validation
- [x] Both defects reproduced by a failing test, then fixed

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: where to ask Git whether a path is a gitlink, given that asking per
  path is expensive and asking inside the classifier breaks its purity.
- Options considered: (a) call Git inside `reviewBundlePathClass`; (b) call it
  per path at the call site, before classification; (c) resolve every candidate
  path in one call before the loop.
- Decision: (c).
- Rationale: (a) makes a pure predicate depend on ambient repository state and
  breaks its fixtures. (b) is correct but turns a bundle over a large change set
  into one process launch per path. (c) is a single `git ls-files --stage --`
  over the subject's own paths, so the cost is bounded by what is being reviewed
  and the classifier stays pure.
- Consequences: the map is built even when no submodule exists, at the cost of
  one Git call per bundle.

### Decision 2
- Date: 2026-09-08
- Context: the original fix ran the lookup only when the path was unclassified,
  which reads as a cheap fast path but encodes the wrong precedence.
- Decision: the index wins over the path's shape.
- Rationale: a path's shape is a heuristic about what a file probably is; the
  index is a fact about what it *is*. When they disagree the fact should win,
  and a gitlink under `src/` is still a gitlink. The original ordering only
  looked correct because the motivating repository vendored at the root.

---

## 6. Validation

### Strategy
Reproduce each defect with a failing test before fixing it, and assert the fix
in both directions so neither test can pass by observing nothing.

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
- Notes: the submodule defect was reproduced before being fixed, with the
  fixture failing on `review subject path api/dep cannot be read`. Making the
  fixture a genuine nested repository — rather than a bare
  `update-index --cacheinfo`, which leaves the path added in the index and
  absent on disk — surfaced two further truthful blockers in sequence
  (`git status AD`, then `A`), both resolved by committing the gitlink as a real
  checkout would have it. Restoring the `class == ""` guard and disabling the
  component filter each turn their test red; all eight packages pass with both
  in place.

### Results summary
- Successes: R1, R2, R3, R4 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <review_bundle.go — the gitlink map is consulted before the shape class is honoured; TestReviewBundleClassifiesSubmoduleUnderAMappedComponent covers a submodule under a mapped component>
- R2 [satisfied] <reviewBundleGitlinks issues one `git ls-files --stage --` over the subject's candidate paths, replacing the per-path lookup>
- R3 [satisfied] <review_plan.go — profileEvidence filters overlays by the components their selection matched; TestReviewPlanScopesOverlayEvidenceToItsMatchedComponent asserts each component keeps only its own overlay's classes>
- R4 [satisfied] <the component=="" branch skips the filter, so the repository-wide tool retains the union>

### Known gaps
- A profile selection carries the components it matched, so the scoping is only
  as accurate as that record. A selector that matches nothing still contributes
  its classes to the repository-wide tool, which is intended but means an
  unused overlay can widen what that tool accepts.

---

## 7. Final Report

### Follow-ups

- [open] Review whether other synthesised tools should be component-scoped like validate, rather than only the one this defect exposed — owner:unowned crit:low review:2026-12-08
