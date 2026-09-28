# Review archival with joint commit attribution

Date: 2026-09-28. Scope: pose-review-archive-coattributed-fragments.

## Reproduction and cause

On 11d1630, the published reader blocks spec:pose-validate-report-carries-its-run on the pose-one-follow-up-format fragment, archived in v5.0.2. Squash commit 7859fae11d745c772fbdaee5b40a2a27a549f396 contains both POSE-Spec lines and both fragment paths. The reader had supplied the primary spec as the only eligible ledger owner.

The new shared-commit regression fails before the fix with the same owner error. Disjoint-commit, different-path and empty-attribution cases already fail closed.

## Fix and validation

A missing pending fragment can resolve through an alternate owner only when the integrity graph proves a shared immutable attributed commit, matching observed creation/modification and a Git diff-tree proving the pending path changed in that shared commit. Overlap elsewhere in an aggregated change set grants no ownership. The existing ledger still checks exact owner, unique release, frozen digest, committed manifest/archive and root containment. Subject identity keeps the original pending path.

- go test ./internal/pose -run ReviewReleaseArchive -count=1: PASS, including the red/green shared-commit regression and existing tamper negatives.
- Candidate build: PASS.
- Candidate review bundle spec:pose-validate-report-carries-its-run --json: prepared, zero blockers, 22 subject entries. The unmodified reader fails on the same graph.

Native module matrix: 34/34 registered checks pass (49.4 seconds). The initial sandbox run was blocked by local HTTP sockets; rerunning with approved local sockets passed. Governed review/closeout are the remaining completion gates. This development correction does not alter the published v6.0.0 tag, archive or release ledger.
