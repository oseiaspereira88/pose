---
name: pose-release-closeout
description: Use to prepare, publish, reconcile and independently verify a POSE release without mutable notes, tag overwrite or fabricated provider state. Trigger keywords - release, publish, tag, changelog cut, release closeout, version cut.
when_to_use: Reviewed specs and roadmap are terminal and a new immutable POSE version must be cut and published.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, release-write
---

# Skill: pose-release-closeout

## Required reading

1. [Release workflow](../../../.pose/workflows/release.md).
2. [Release integrity rule](../../../.pose/rules/release-integrity.md).
3. [Release policy](../../../.pose/release-policy.json).

## Procedure

1. Require terminal reviewed delivery scope and an explicit target version.
2. Plan, then prepare with explicit apply; review and commit the frozen snapshot.
3. Run strict release, compatibility, security and full validation gates.
4. Require a clean worktree and absent tag; never stage broadly or force.
5. Publish the new tag and monitor provider completion.
6. Import retained publication evidence and independent verification evidence.
7. Do not stop at tag creation: terminal success requires the policy's verified
   projection. On failure, record it and preserve the immutable tag.

## Output requirements

- Immutable manifest, notes and archived fragments.
- New non-overwritten tag bound to the reviewed commit.
- Provider evidence with minimized identity and asset digests.
- Independent verification bound to that publication.
- Empty post-cut pending queue except genuinely newer work.
