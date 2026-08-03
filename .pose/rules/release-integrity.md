# Rule: Release integrity

## Required guarantees

- Pending fragments are candidates, never proof of a release.
- A tagged publication consumes only committed prepared notes and fragments.
- Tags, publication and verification are distinct facts with confined typed
  evidence; verification binds the exact publication and asset digests.
- Release manifests, notes and archived fragments are immutable after cut.
- Missing policy after adoption, duplicate fragments, stale snapshots, tag
  drift or stronger-than-evidence states block strict release claims.
- Release automation may not use broad staging, force-tagging or force-pushing.
- Historical backfill reports confidence and gaps without manufacturing facts.

## Minimum gates

Run `pose release check --version vX.Y.Z --strict`, compatibility and full
module validation before tagging. Require `pose release status` to project
`verified` before declaring publication complete.
