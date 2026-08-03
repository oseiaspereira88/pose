# Workflow: Evidence-backed release

## Objective

Consume reviewed unreleased fragments into one immutable candidate and keep
preparation, tagging, provider publication and independent verification as
separate evidence-backed facts.

## Steps

1. Confirm the roadmap and member specs are terminal with current macro review.
2. Update and review the authoritative project version.
3. Run `pose release plan --version vX.Y.Z`; resolve every blocker.
4. Run `pose release prepare --version vX.Y.Z --apply` and version only the
   manifest, archived fragments and canonical notes it creates.
5. Run `pose release check --version vX.Y.Z --strict`, full validation,
   compatibility, signing/SBOM and independent-verification gates.
6. Require a clean worktree and create a new annotated tag without overwrite or
   force. Tagged CI must consume `pose release notes --version vX.Y.Z`.
7. Import retained provider evidence with `pose release record`; do not infer
   publication from tag presence.
8. Import independent verification bound to the publication and asset digests.
9. Require `pose release status --version vX.Y.Z` to report `verified` before
   planning the next development version with `pose release open-next`.

## Failure handling

A failed provider run leaves the release tagged but unpublished. Record a
`failed` event, repair the workflow, and rerun against the same immutable tag;
never recreate or force-push it. A yank is an explicit event and does not erase
prior publication history.
