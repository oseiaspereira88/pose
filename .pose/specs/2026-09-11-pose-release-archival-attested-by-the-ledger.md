---
slug: pose-release-archival-attested-by-the-ledger
status: in-progress
created_at: 2026-09-11
completed_at:
supersedes:
depends_on: pose-release-cycle-debt-closure
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: Release archival is attested by the ledger, never written into specs

## 1. Intent

### Goal
A spec that declared its changelog fragment shall pass `pose artifact-check
--strict` after the release that archives the fragment. The release shall not
edit the spec to get there.

### Business value
`pose release prepare` rewrites each released spec's
`created: .pose/changelogs/unreleased/X` claim into
`renamed: .pose/changelogs/unreleased/X -> .pose/changelogs/vN/X`. No change set
attributed to the spec performs that rename, so every released spec fails
`artifact-check` with `action-mismatch`: 52 of the 54 specs carrying the
rewritten claim on `main` at 3a99551. The rewrite lands in the Technical Plan,
which the sealed review bundle digests, so a spec closed before the cut has its
approved subject changed by the cut. Cutting 5.0.5 moved the bundle digest of
`pose-manual-merge-backs-up-only-local-edits` from `ea372ec1…` to `4932a5a0…`.

Found by the review of the v5.0.5 release PR (pose#104). The decision, and the
six options it rejects, is recorded in ADR
`2026-09-11-release-archival-is-attested-by-the-ledger-never-written-into-specs`.

### Constraints
- No existing spec is edited: rewriting the 54 claims would repeat the defect.
- Declaration, Git observation and ledger attestation stay distinct witnesses,
  as the delivery integrity ADR requires.
- The release manifest schema does not change.

### Non-goals
- Requiring member specs to be terminal in `release check --strict`. That is a
  separate decision, which this spec makes possible.

---

## 2. Requirements

### Functional
- R1: `pose release prepare` shall leave every spec file byte-identical.
- R2: A `created` or `modified` claim on `.pose/changelogs/unreleased/X` that is
  no longer tracked shall pass existence when three conditions hold: a release
  manifest lists fragment `X` for the same spec; the archived path
  `.pose/changelogs/<version>/X` is tracked; and the archived file still has
  the digest the manifest froze.
- R3: A `renamed: .pose/changelogs/unreleased/X -> .pose/changelogs/vN/X` claim
  written by an earlier release shall pass the action check under the same
  conditions, with the manifest of `vN`.
- R4: Any other case shall stay a finding, as today: a fragment another spec
  owns, one no manifest lists, one whose archived content changed, or any other
  path. Its message shall say what the manifests showed.
- R5: The delivery integrity graph shall record each archival it used as edges of
  its own, separate from change-set edges. The declared artifact is
  `archived-by` a `release:<version>` node, which `archives` the file at its new
  path. The archived path shall count as claimed by the spec.
- R6: `pose artifact-check` shall say, in its text output, which claims resolved
  through a release, and where the file went.

### Non-functional
- A repository with no release manifests builds the same graph as before,
  digests included.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/release_archive.go` — reading what the manifests attest
- `pose-mcp/internal/pose/delivery_integrity.go` — resolving claims through it
- `pose-mcp/internal/cli/artifact_integrity.go`, `surface_check.go` — passing it in
- `pose-mcp/internal/cli/release_lifecycle.go` — prepare stops editing specs

### Artifacts
- created: .pose/specs/2026-09-11-pose-release-archival-attested-by-the-ledger.md
- created: .pose/changelogs/unreleased/pose-release-archival-attested-by-the-ledger.md
- created: pose-mcp/internal/pose/release_archive.go
- created: pose-mcp/internal/pose/release_archive_test.go
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

The ADR is not claimed here: it was created before this spec existed, in a
commit outside its change set, and `.pose/adr` is not a governed root. Decision 1
names it.

### Technical risks
- The graph's input digest must not change for a repository with no releases,
  or every instance's index churns on upgrade for nothing. The archivals enter
  the digest only when there are any.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Read what each manifest attests, and resolve claims through it (R2, R3, R4, R5)
- [x] Increment 2: Prepare stops editing specs (R1)
- [x] Increment 3: Say it in `artifact-check` and the manuals (R6)

### Validation
- [x] On this repository, the released specs' `action-mismatch` findings resolve
  with no spec edited

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: the defect could be fixed in `release prepare`, in the checker, or in
  both; seven options are weighed in the ADR.
- Decision: specs stay untouched, and the release manifest attests the archival
  — ADR `2026-09-11-release-archival-is-attested-by-the-ledger-never-written-into-specs`.
- Rationale: it is the only option that fixes `artifact-check` without the cut
  changing a sealed subject, and without a manual step per spec per release.

### Decision 2
- Date: 2026-09-11
- Context: with `.pose/changelogs` governed, a rename claim an earlier release
  wrote left the spec's own creation of the fragment undeclared: the claim names
  the rename, the change set shows the creation. The old engine had the same
  gap; this repository does not govern the path, so it never showed.
- Decision: an observed creation of a pending fragment is explained by a
  declared rename the ledger attests.
- Rationale: R3 promises the rewritten claim passes; the creation is the half of
  that rename the spec did perform, so leaving it undeclared would keep the claim
  failing in any repository that governs its changelogs.

---

## 6. Validation

### Strategy
Cut a release in a fixture and check the released spec before and after. Then
measure this repository's own graph.

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
- Date: 2026-09-11
- Environment: local, Go 1.26
- Notes: TestAReleasedSpecStillPassesArtifactCheckAfterTheCut commits a spec
  with its trailer, cuts a release in the fixture, commits the cut without one,
  and requires `artifact-check --strict` to pass and name where the fragment
  went; with the ledger disabled it fails with the existence finding, and the
  rewritten-claim test with the reported `action-mismatch`. Measured on this
  repository in a single-branch clone with the source-built binary:
  `action-mismatch` findings fall from 81 to 30, the graph lists 52 archivals
  with 104 edges, and `artifact-check --strict` passes for the specs released
  in 5.0.4 and 5.0.5, which failed before.

### Results summary
- Successes: R1–R6 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestReleasePrepareLeavesEverySpecByteIdentical requires both specs byte-identical after prepare, and the fragment archived>
- R2 [satisfied] <TestAnArchivedFragmentClaimResolvesThroughTheRelease; TestAReleasedSpecStillPassesArtifactCheckAfterTheCut passes a real cut end to end>
- R3 [satisfied] <TestARenameAnEarlierReleaseWroteResolvesThroughItsManifest, including a governed changelog root and a manifest of another version; TestASpecAnEarlierReleaseRewroteStillPasses>
- R4 [satisfied] <TestOnlyAnArchivalTheReleaseAttestsResolvesAClaim: another spec, no manifest, edited archive, untracked archive each name what the manifests showed; a non-fragment path keeps its exact message>
- R5 [satisfied] <archived-by and archives edges and the reverse entry asserted; no changes edge to the archive; no orphan for a governed archived fragment in the end-to-end test>
- R6 [satisfied] <the end-to-end test requires artifact.archived=<pending> -> <archived> (release <version>)>

### Known gaps
- Two claims from v0.16.2 and v0.16.3 still fail: they were repointed by hand
  to the archive path before release manifests existed, so no manifest attests
  them. Unchanged by this spec, and outside what the ledger can speak for.

---

## 7. Final Report

### Summary
A release no longer edits the specs it ships, and a released spec passes
`artifact-check` through what the release manifest attests.

### Follow-ups

- None.
