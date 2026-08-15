---
slug: pose-review-scope-trailer-check
status: draft
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-mcp, cli
delivers:
---

# Spec: pose-review-scope-trailer-check

---

## 1. Intent

### Goal
Make `pose doctor` (or an equivalent early check) detect and warn when a spec
that declares `delivers:` has no commit in its attributable history carrying
the `POSE-Spec: <slug>` trailer, instead of letting the failure surface only
much later as an opaque `no immutable attributed change set exists for
spec:<slug>` when someone tries to seal a review bundle.

### Business value
Surfaced in `github.com/oseiaspereira88/pose#17` (closed, scope: root
contamination and stdio approval messaging). The issue's second and third
comments found a second, unrelated root cause for the review-bundle blocking
reported in its title: `resolveGitChangeSet`
(`pose-mcp/internal/cli/artifact_integrity.go:68`), when called without
explicit `--from`/`--to`, falls back to scanning up to 500 commits for one
carrying the trailer `POSE-Spec: <spec>` (`commitsWithSpecTrailer`) and fails
with `no commits carry POSE-Spec: %s` if none exists — a local
adoption/practice gap, not a code defect, but one the current error message
does not point back to. In the reporter's repository at the time, 4 of 6
specs declaring `delivers:` had zero commits with the trailer. This repeats
for any project that adopts POSE review bundles without also adopting the
trailer convention in its commit messages.

### Constraints
- Detection only; do not auto-insert trailers into existing commit messages
  (rewriting history is out of scope and dangerous).
- Must not require network access or any provider-specific integration —
  stays inside the existing offline `pose doctor`/local git model.
- Should not fire for specs that legitimately close via explicit
  `--change-from`/`--change-to` (or `pose artifact-check --from/--to`)
  without ever relying on trailer auto-discovery — the absence of a trailer
  is not itself wrong; it only matters when it is the *only* mechanism
  available for a scope that has not been closed some other way.

### Non-goals
- Making `resolveGitChangeSet`'s trailer scan itself smarter (e.g. searching
  more than 500 commits, fuzzy matching) — this spec is about warning before
  the point of failure, not changing the resolution algorithm.
- A commit-msg hook or CI gate that enforces the trailer on every commit —
  worth considering later, out of scope here.

---

## 2. Requirements

### Functional
- R1: When a spec has `delivers:` populated and `status: in-progress` or
  `status: done`, and no commit in `git log --all` carries the trailer
  `POSE-Spec: <slug>`, and the spec has no persisted change set in
  `.pose/reports/history/*.jsonl` recording an explicit `--change-from`/
  `--change-to` range, `pose doctor` shall warn that sealing a review bundle
  for this scope will fail trailer auto-discovery, and shall point to
  `pose report --spec <slug> --change-from <rev> --change-to <rev>` as the
  explicit-range alternative.
- R2: The warning message shall name the exact failure it prevents
  (`no immutable attributed change set exists for spec:<slug>`) so it is
  discoverable by someone who already hit that error and searches for it.

### Non-functional
- Must not meaningfully slow `pose doctor` on repositories with long git
  history (bound the same way `commitsWithSpecTrailer` already bounds itself,
  `--max-count=500`, or cheaper).

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go`
- `pose-mcp/internal/cli/artifact_integrity.go` (read-only reuse of
  `commitsWithSpecTrailer`, or an exported variant)

---

## 4. Tasks

### Planning
- [x] Capture the finding from `github.com/oseiaspereira88/pose#17`
      (comments 2-3) as this spec
- [ ] Decide the exact `pose doctor` check shape and message wording
- [ ] Confirm how to detect "already closed via explicit range" without
      false-positiving on specs closed the way this session closed
      `pose-unified-review-convergence` and
      `pose-review-bundle-root-file-classification` (explicit
      `--change-from`/`--change-to`, no trailer)

### Implementation
- [ ] Implement incrementally

### Validation
- [ ] Run the mandatory checks

---

## 6. Validation

### Strategy
Unit tests in `pose-mcp/internal/cli` covering: a spec with a trailer commit
(no warning), a spec closed via an explicit persisted change set (no
warning), and a spec with neither (warning fires with the exact scope slug
and the pointed-to remediation command). Exercised against `pose doctor`'s
existing test fixtures rather than a new harness.

---

## 7. Final Report

### Follow-ups
- [open] Consider a `pose init`/`pose new-spec` prompt or `AGENTS.md` note
  reminding authors to use the `POSE-Spec: <slug>` trailer, so adoption
  happens before the gap is hit rather than after (owner:@pose-maintainers
  crit:low review:2026-11-15)
