# ADR: Release archival is attested by the ledger, never written into specs

## Status
Proposed (2026-09-11) — awaiting the maintainer's decision; to be implemented by
a spec once accepted. Extends the delivery integrity graph ADR
(`2026-08-02-delivery-integrity-graph-and-git-observed-provenance`) with one
witness, and replaces the mechanism — not the requirement — of
`pose-release-cycle-debt-closure` R2.

## Context

A spec declares the changelog fragment it adds, as the specs in this repository
do by convention: `- created: .pose/changelogs/unreleased/<slug>.md`. `pose
release prepare` then archives the fragment to
`.pose/changelogs/<version>/<slug>.md`, as the release ledger ADR (ADR-018)
requires.

Every artifact claim answers to two checks in the delivery integrity graph:

- **existence** — the path the claim names now is tracked at the selected head;
- **action** — the claimed action appears in the Git change sets attributed to
  the spec.

After a cut, no claim a spec can hold passes both. `created: …/unreleased/X`
fails existence, because the file moved. `renamed: …/unreleased/X ->
…/vN/X` fails action, because the move happens in the release commit, and that
commit is not attributed to the spec. `created: …/vN/X` fails action too: the
spec's own commits created the file somewhere else.

`pose-release-cycle-debt-closure` R2 fixed the first failure by having `release
prepare` rewrite the claim into the rename (`repointFragmentClaims`). That traded
an existence error for an action error, and moved the damage to a check that
`pose check --strict` does not gate but `pose artifact-check --strict` does.
Measured on `main` at 3a99551:

- 54 specs carry a rewritten `renamed: .pose/changelogs/unreleased/…` claim.
- 52 of them fail `artifact-check` with `action-mismatch` on that claim: 52 of
  the repository's 81 `action-mismatch` findings.
- The other two pass only because someone recorded the v1.1.0 release commit as
  an explicit range for each of them (`range:0d40bc7^..0d40bc7`). That manual
  step was never repeated after v1.1.0.

The rewrite has a second effect, and it is worse than the first. The Artifacts
list sits in the spec's Technical Plan, and the sealed review bundle digests that
section as semantic review input. Cutting 5.0.5 changed the Technical Plan digest
of `pose-manual-merge-backs-up-only-local-edits` from `f69e228c…` to `33f4801f…`,
and its bundle digest from `ea372ec1…` to `4932a5a0…`. A spec closed before the
cut therefore has its approved subject rewritten by the cut. That is the failure
the sealed-bundle ADR exists to prevent: an approved subject changed by something
that happened after the approval. It also defeats step 1 of the release
workflow, which asks for member specs to be terminal, with current review,
*before* prepare: doing that step correctly is what the next step then
invalidates. No spec released since v2.0.0 was closed before its cut.

## Options considered

1. **Keep the rewrite.** Rejected: it fails the action check on every released
   spec, and it rewrites the sealed subject of any spec closed before the cut.
2. **Put a `POSE-Spec:` trailer for the released specs on the release commit.**
   Rejected, measured: in a single-branch clone of the v5.0.5 release branch
   with the trailer added, `artifact-check` still reports the `action-mismatch`,
   because the attributed range both creates and moves the fragment, so Git sees
   it created under `v5.0.5/`. The range also adds `undeclared` errors for the
   version-bump surfaces (`docs-site/docs/ci.md`,
   `pose-mcp/internal/version/version.go`). A release with several specs would
   need several trailers on one commit, which range resolution cannot express
   without contaminating every range involved.
3. **Record the release commit for each spec with `pose report
   --change-from/--change-to` after the cut.** This is what makes the two v1.1.0
   specs pass. Rejected: it is a manual step per spec per release. Since 5.0, the
   release commit also carries governed version-bump changes, which each feature
   spec would then have to claim as its own.
4. **Keep the rewrite, and let the ledger satisfy the rename in the action
   check.** Rejected: it fixes `artifact-check`, but the cut still edits the
   Technical Plan, so the conflict with sealed review stays.
5. **Leave consumed fragments in `unreleased/` and derive the pending queue from
   the manifests.** Rejected: claims would stay valid without a new witness, but
   it changes ADR-018's archive layout and every reader of `unreleased/`, and
   the directory grows forever.
6. **Stop declaring fragments as artifacts.** `.pose/changelogs` is outside the
   governed roots, so nothing requires the claim, and the link from fragment to
   spec already lives in the fragment's frontmatter and in the manifest.
   Rejected: the 54 existing claims would still need handling, and it removes a
   declared link instead of explaining it.
7. **A spec's claims are its author's alone. The release ledger attests the
   archival as a witness of its own.** Selected.

## Decision

**A spec's artifact claims are changed only by its author.** `pose release
prepare` no longer edits spec files. `repointFragmentClaims` is removed, and so
is the part of prepare's rollback that restores specs.

**The release manifest attests one fact, and nothing else:** fragment `X` of spec
`S` was archived at `.pose/changelogs/<version>/X`. The delivery integrity graph
records this as its own edge, from the artifact to a `release:<version>` node.
It is never recorded as a Git change observed for the spec. Declaration, Git
observation and ledger attestation stay three distinct witnesses, as the
delivery integrity ADR requires.

**Resolving a fragment claim.** A `created` or `modified` claim on
`.pose/changelogs/unreleased/X` that is no longer tracked passes existence when
all three of these hold:

- some release manifest lists fragment `X` for the same spec;
- `.pose/changelogs/<version>/X` is tracked at the selected head;
- its content still has the digest the manifest froze (`ReleaseDigest` of the
  raw file, which is how prepare computed it).

The action check is unchanged. The spec's own change set created the fragment,
and it still does.

**Claims already rewritten.** A `renamed: .pose/changelogs/unreleased/X ->
.pose/changelogs/vN/X` claim passes when manifest `vN` lists `X` for that spec,
under the same conditions. Nothing rewrites the 54 existing specs. Rewriting
them would repeat the defect, and for a sealed spec it would redefine the
subject a second time. New cuts simply stop producing such claims.

**Bounded to fragments.** Only paths under `.pose/changelogs/unreleased/` that a
manifest lists for the same spec are resolved this way. The following stay
findings, as they are today, with a message naming the manifest that was
consulted:

- a fragment the manifest lists for another spec;
- a fragment no manifest lists;
- an archived fragment whose digest changed;
- any other path.

## Consequences

- Positive: released specs pass `artifact-check --strict` with no manual step,
  and the 52 current errors resolve without editing any spec. Measured: each of
  the 52 fragments is listed by its version's manifest for the same spec, and
  each archived file still has the digest that manifest froze.
- Positive: a release cut no longer changes any spec's sealed subject. Closing
  specs before the cut, as workflow step 1 asks, becomes possible. Whether
  `release check --strict` should *enforce* terminal member specs is a separate
  decision, and it is not taken here; this decision is its precondition.
- Positive: prepare's transaction gets smaller. It moves fragments and writes the
  manifest and notes, and it touches no spec.
- Trade-off: a claim can now resolve to a path it does not name. The graph and
  the `artifact-check` output must say so — "archived by release vN at …" — or
  the resolution reads as magic.
- Trade-off: the delivery integrity graph gains a node type and an edge type, and
  a new input: `.pose/releases/*/manifest.json`. The change is additive and
  schema-versioned, with golden fixtures updated. Manifests are immutable under
  ADR-018, so the new input causes no freshness churn.
- Neutral: `TestReleasePrepareRepointsConsumedSpecArtifactClaims` asserts the
  behaviour being removed. It is replaced by a test that prepare leaves every
  spec byte-identical, and that `artifact-check` passes for the released spec
  after the cut.
- Neutral: instances that already cut releases are covered by the rule for
  rewritten claims. Nothing needs migrating.

## Review triggers

Revisit this decision:

- if an archived fragment ever has to move again, for example through a yanked
  or re-cut version;
- if the engine starts performing other moves of declared artifacts. The rule
  generalises: an engine-performed move is attested by a record the engine
  keeps, never written into the declaration.
