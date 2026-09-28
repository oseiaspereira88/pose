---
slug: pose-review-archive-coattributed-fragments
status: in-progress
created_at: 2026-09-28
completed_at:
depends_on: pose-release-archival-attested-by-the-ledger
priority: 0
components: pose-mcp
task_type: bugfix
---

# Spec: Resolve archived review fragments co-attributed in a shared commit

## 1. Intent

Unblock a legitimate review subject containing another spec's archived fragment when the same immutable Git commit attributes that path to both specs. Reuse the existing ledger witness without inferring ownership from dependency declarations or filenames.

At source HEAD 11d1630, review of pose-validate-report-carries-its-run fails on the archived pose-one-follow-up-format fragment. Both POSE-Spec lines are present in squash commit 7859fae11d745c772fbdaee5b40a2a27a549f396. Subject reading passes only the primary spec to the archival resolver.

Reuse knowledge:adr-sealed-review-bundles-review: semantic identity must survive release archival while unproved content remains fail closed.

## 2. Requirements

- R1: Before and after committed archival, a shared-commit review subject shall have identical digests when the ledger names a co-attributed path owner.
- R2: Accept a different ledger owner only when a graph change set for that owner shares an immutable attributed commit with the selected set and observes the same fragment path.
- R3: A dependency without shared attribution, a disjoint commit, an unrelated path, wrong owner, duplicate release, dirty/untracked manifest or archive, changed digest and root escape shall remain blocked.
- R4: The real previously blocked scope shall prepare successfully with the candidate CLI; public 6.0.0 bytes and ledger remain immutable.

## 3. Technical Plan

### Artifacts
- created: .pose/specs/2026-09-28-pose-review-archive-coattributed-fragments.md
- created: .pose/changelogs/unreleased/pose-review-archive-coattributed-fragments.md
- created: .pose/reports/2026-09-28-review-archive-coattribution.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/technical-debt.json
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_release_archive_test.go

Keep direct owner resolution unchanged. Supply alternate owners proven by co-attributed immutable commits and matching observed paths. Resolve each through the existing committed, intact and unique archival witness. No release schema or public CLI change.

## 4. Tasks

- [x] Reproduce the real failure and isolate the joint-commit owner mismatch.
- [x] Prove shared-commit regression red before the fix.
- [x] Implement minimal owner selection and preserve negative cases.
- [ ] Run native registered archive integration, module matrix, real preparation, review and closeout.

## 5. Decisions

Accept co-attribution as a Git fact; dependency membership grants no ownership. Reuse frozen archive bytes and keep the attributed pending path as subject identity. Roll back by reverting the minimal subject reader change.

## 6. Validation

### Strategy
Add a shared-commit fixture to the existing release-review suite; require an unchanged subject. Add disjoint-commit and wrong-path negatives; retain all existing owner, digest, ledger and symlink negatives. The registered release-review-archive-integration check runs these cases. Run the module's native matrix and the real scope preparation using the candidate build.

### Requirement trace
- R1 [satisfied] test:TestReviewReleaseArchiveCoattributedFragment shared-commit case fails before the fix and passes with equal subjects after archival.
- R2 [satisfied] test:TestReviewReleaseArchiveCoattributedFragment disjoint commits, different paths and missing commit attribution remain blocked.
- R3 [satisfied] test:TestReviewReleaseArchiveRejectsUnattestedContent existing owner, ledger, digest, duplicate and root escape negatives pass.
- R4 [satisfied] report:.pose/reports/2026-09-28-review-archive-coattribution.md real scope prepares with zero blockers under the candidate CLI.

## 7. Final Report

### Follow-ups
None introduced. Release of this correction follows the canonical POSE mechanism closure plan.
