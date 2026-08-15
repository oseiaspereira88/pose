---
slug: pose-review-bundle-root-file-classification
status: done
created_at: 2026-08-15
completed_at: 2026-08-15
supersedes:
depends_on:
priority: 2
components: pose-mcp
delivers: capability:pose-mcp
---

# Spec: pose-review-bundle-root-file-classification

---

## 1. Intent

### Goal
Teach `reviewBundlePathClass` to classify the two well-known repository-root
release files (`README.md`, `compatibility.json`) so review bundles whose
change set touches them can seal instead of failing closed as unclassified.

### Business value
Discovered while retroactively closing `pose-unified-review-convergence`:
its implementation shipped bundled inside `chore(release): prepare v1.2.0`
(commit 422907c), a commit that also carries the ordinary release version
bump touching `README.md` and `compatibility.json`. Any review bundle whose
attributed change set includes that commit inherits those two paths and
cannot seal, because the classifier has no rule for them — even though they
are known, low-risk, well-understood files, not opaque unknown content.

### Constraints
- Preserve the fail-closed guarantee for genuinely unknown/opaque paths
  (`TestReviewBundleRejectsUnclassifiedSubjectPath` must keep failing).
- Follow the exact classification pattern already used for `POSE.md`/
  `AGENTS.md` (documentation) and `.pose/adr|knowledge|changelogs/`
  (governance) in `reviewBundlePathClass`.

### Non-goals
- Classifying arbitrary root-level files generically (e.g. by extension or
  directory depth) — only the two specific, known files are added.
- Changing how `pose report`/`pose index` capture change-set paths.

---

## 2. Requirements

### Functional
- R1: When a review bundle's attributed change set includes `README.md`,
  `reviewBundlePathClass` shall classify it as `documentation` and include it
  in the review subject, instead of failing closed as unclassified.
- R2: When a review bundle's attributed change set includes
  `compatibility.json`, `reviewBundlePathClass` shall classify it as
  `governance` and include it in the review subject, instead of failing
  closed as unclassified.

### Non-functional
- No behavior change for any other path, including genuinely unrecognized
  paths (fail-closed regression coverage must keep passing).

### Compatibility
- Purely additive classification rule; no schema or CLI surface change.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go`

### Artifacts
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: two exact-name string comparisons added to an existing exhaustive
  if-chain; existing fail-closed test pins the negative case.

---

## 4. Tasks

### Planning
- [x] Confirm intent (discovered while closing pose-unified-review-convergence)

### Implementation
- [x] Add `README.md` and `compatibility.json` to `reviewBundlePathClass`

### Validation
- [x] `TestReviewBundleClassifiesRootReleaseFiles` (new)
- [x] `TestReviewBundleRejectsUnclassifiedSubjectPath` (existing, still fails closed)
- [x] `go -C pose-mcp test ./...`, `go -C pose-mcp vet ./...`

---

## 6. Validation

### Strategy
Unit-level regression in `pose-mcp/internal/pose`, plus the full module test
suite to guard against unrelated regressions.

### Requirement trace
- R1 [satisfied] `TestReviewBundleClassifiesRootReleaseFiles`
- R2 [satisfied] `TestReviewBundleClassifiesRootReleaseFiles`

### Known gaps
- None. The fix is intentionally narrow (two named files); a broader
  root-file classification rule is not in scope.

---

## 7. Final Report

### Delivered scope
Extended `reviewBundlePathClass` so `README.md` classifies as `documentation`
and `compatibility.json` classifies as `governance`, both included in the
review subject. Fail-closed behavior for unknown paths is unchanged and
still covered by `TestReviewBundleRejectsUnclassifiedSubjectPath`.

### Files and modules changed
- `pose-mcp/internal/pose/review_bundle.go`: two new exact-path branches.
- `pose-mcp/internal/pose/review_bundle_test.go`:
  `TestReviewBundleClassifiesRootReleaseFiles`.

### Validation executed
- `go -C pose-mcp test ./internal/pose/... -run ReviewBundle`: SUCCESS.
- `go -C pose-mcp test ./...`: SUCCESS (pose-mcp package suite).
- `go -C pose-mcp vet ./...`: SUCCESS.

### Residual risks
- None identified; change is additive and narrowly scoped.

### Follow-ups
- [open] If more root-level release/governance files accumulate the same
  unclassified gap, consider a small allowlist config instead of extending
  the if-chain per file (owner:@pose-maintainers crit:low review:2026-11-15)
