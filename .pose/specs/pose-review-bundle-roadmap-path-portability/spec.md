---
slug: pose-review-bundle-roadmap-path-portability
status: done
created_at: 2026-08-15
completed_at: 2026-08-15
supersedes:
depends_on:
priority: 1
components: pose-mcp
delivers: capability:review-bundle-roadmap-path-portability
---

# Spec: pose-review-bundle-roadmap-path-portability

---

## 1. Intent

### Goal
Stop `reviewBundleScopeProjection`'s `"roadmap"` case from keying its
semantic section on an absolute filesystem path, so a sealed roadmap
review bundle reads as fresh from any checkout of the same content.

### Business value
`github.com/oseiaspereira88/pose#26`: discovered while publishing v1.3.0.
Closed the `adaptive-instance-provisioning` roadmap and sealed its review
bundle locally; CI immediately failed `pose check --strict` with
`resolve closeout blockers for roadmap:adaptive-instance-provisioning`, on
the exact same commit that passed locally. Reproduced with a plain
`git clone` of that commit to a different path (no CI, no shallow clone
involved) — `pose check --strict` failed identically, isolating the cause
to path, not environment.

Root cause: `reviewBundleScopeProjection` (`review_bundle.go`) recorded the
roadmap section's `Path` as `filepath.ToSlash(rm.Path)`, where `rm.Path` is
`Store.Root` joined with the roadmap file — an absolute path. Two
checkouts of identical content at different absolute paths (a developer's
machine and CI, or CI and any fresh clone) compute different
`ChangedSections` keys, so the roadmap reads as "changed"
(`state: superseded`) on every checkout other than the one that sealed it.
Every other review-bundle scope kind already avoids this: `spec` keys
sections by heading name, `milestone` by the milestone ID string. This is
very likely the first roadmap ever sealed under the review-bundle
mechanism (`.pose/policy/review.json`'s `review_bundles_adopted_at:
2026-08-14`, one day before this was found), which is why it went
undetected until now.

### Constraints
- Fix must not change the digest/content identity of the roadmap section
  (`Digest: digestText(normalizeBundleText(rm.Body))`) — only the `Path`
  key, which was never meant to carry filesystem-location semantics in the
  first place (compare `milestone`'s `Path: scope.Milestone`, an ID, not a
  path).
- Must not affect `spec`/`milestone` scope projection — confirmed those
  already use non-absolute keys; this is a `roadmap`-scope-only fix.

### Non-goals
- Auditing every other potential absolute-path leak across the codebase —
  scoped to the one confirmed instance blocking this release. `rm.Path`
  (spec lifecycle exclusion, line ~274) also carries an absolute path but
  feeds the `excluded` list, not `Sections`, so it never enters the
  freshness diff — noted as a residual finding, not fixed speculatively
  here (see Follow-ups).

---

## 2. Requirements

### Functional
- R1: `reviewBundleScopeProjection`'s `"roadmap"` case shall key the
  roadmap's semantic section on a repo-relative path
  (`.pose/roadmaps/<slug>.md`), not `rm.Path`.
- R2: sealing a roadmap review bundle from one checkout and verifying it
  from a different checkout of the identical commit shall report the
  roadmap as fresh (no spurious `ChangedSections` entry).

### Non-functional
- No change to `spec`/`milestone` scope projection behavior.

### Compatibility
- **Breaking for the one roadmap bundle already sealed under the old
  (buggy) key** — `adaptive-instance-provisioning`'s roadmap-scope bundle
  must be resealed after this fix lands, since the section key it was
  computed against no longer exists. This is the same "reseal after
  changing the mechanics" pattern already established throughout this
  session's closeouts, not a new kind of migration.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go`
  (`reviewBundleScopeProjection`, `"roadmap"` case)

### Artifacts
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_bundle_roadmap_path_test.go
- renamed: .pose/changelogs/unreleased/pose-review-bundle-roadmap-path-portability.md -> .pose/changelogs/v1.3.0/pose-review-bundle-roadmap-path-portability.md

### Delivery targets
- capability:review-bundle-roadmap-path-portability module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: single-field change (`Path` key construction) in one scope case;
  confirmed no other code reads `ReviewBundleInput.Path` for the roadmap
  kind expecting a filesystem path (it is only ever used as an opaque
  diffing/display key).

---

## 4. Tasks

### Planning
- [x] Reproduce empirically: `git clone` (both shallow and full) of the
      exact release commit to a different path reproduces the CI failure
      locally, isolating the cause to absolute-path leakage, not CI
      environment or shallow-clone history (issue #26 investigation, this
      repo, 2026-08-15)
- [x] Root-caused to `reviewBundleScopeProjection`'s roadmap case; confirmed
      `spec`/`milestone` cases already avoid the same mistake

### Implementation
- [x] Changed the roadmap section's `Path` to
      `filepath.ToSlash(filepath.Join(".pose", "roadmaps", scope.Slug+".md"))`
      (R1)
- [x] Added `TestReviewBundleRoadmapScopeProjectionIsPathIndependent`:
      seals identical roadmap content from two different `Store.Root`
      temp-dir paths and asserts identical `Path`/`Digest` (R2). Verified
      the test fails against the pre-fix code (`git stash` the fix,
      re-run, confirm failure, restore) before trusting it as a real guard.

### Validation
- [x] `go -C pose-mcp test ./...`, `go vet ./...`, `gofmt -l .`: all clean
- [x] Confirmed the new test fails against the old code and passes against
      the fix (not a vacuous regression guard)

---

## 6. Validation

### Strategy
Direct unit test against `reviewBundleScopeProjection` using two `Store`
instances rooted at different temp directories with identical roadmap
content — the precise shape of the real bug (two checkouts, same
content) — plus verification the test actually fails pre-fix.

### Requirement trace
- R1 [satisfied] `TestReviewBundleRoadmapScopeProjectionIsPathIndependent`
  asserts the computed `Path` never contains either checkout's root.
- R2 [satisfied] same test asserts identical `Path` and `Digest` across
  both checkouts.

### Known gaps
- None identified for the confirmed instance. The `spec` lifecycle
  exclusion's own absolute-path leak (currently harmless, feeds
  `excluded` not `Sections`) is tracked as a follow-up, not fixed here.

---

## 7. Final Report

### Delivered scope
Fixed the one confirmed absolute-path leak blocking the v1.3.0 release:
`reviewBundleScopeProjection`'s roadmap case now keys its semantic section
on a repo-relative path, matching how `spec`/`milestone` scopes already
behave. A sealed roadmap review bundle now reads as fresh from any
checkout of the same content, not only the machine that sealed it.

### Files and modules changed
- `pose-mcp/internal/pose/review_bundle.go`: roadmap section `Path`
  construction.
- `pose-mcp/internal/pose/review_bundle_roadmap_path_test.go`: new
  regression test.

### Validation executed
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l .`: clean.
- Manually confirmed the regression test fails against the pre-fix code.

### Residual risks
- The `adaptive-instance-provisioning` roadmap's existing sealed bundle
  (computed under the old key) needs resealing after this closes — handled
  as part of this session's cascade-reseal step, not a separate follow-up.

### Follow-ups
- [open] `reviewBundleScopeProjection`'s spec-scope `excluded` lifecycle
  entry (`review_bundle.go` line ~274) also carries an absolute
  `filepath.ToSlash(sp.Path)` — currently harmless since `excluded` never
  feeds `ChangedSections`, but the same class of mistake; worth a small
  follow-up for hygiene/defense-in-depth, not urgent since it has no
  observed effect today.
