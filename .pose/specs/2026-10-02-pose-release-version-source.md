---
slug: pose-release-version-source
status: done
created_at: 2026-10-02
completed_at: 2026-10-02
delivers: surface:release-version-source
components: pose-mcp
task_type: feature
priority: 1
---

# Spec: Let a project declare the source of its release version

## 1. Intent

### Goal

Let `.pose/policy/release.json` name the file that holds the project's own
version, so `pose release` can plan, prepare and check a release for a project
that is not the POSE engine.

### Business value

The release lifecycle compares `--version` with the compiled version of the
`pose` binary (`cli/release_lifecycle.go:136-139`, source recorded as
`pose-mcp/internal/version/version.go`). In any other repository this makes every
cut impossible except one whose version equals the engine's. Measured on
2026-10-02 with the installed binary: a fixture repository with an adopted policy
and a valid fragment answered `pose release plan --version v0.2.0` with `target
v0.2.0 differs from authoritative version evidence v6.1.0`. Harne8 holds 204
unreleased fragments from 2026-07-12 to 2026-09-30 and has never cut a release
for this reason; its release policy is still empty.

### Constraints

The engine's own repository keeps working with no policy change, and its sealed
manifests keep verifying. The policy digest recorded in a manifest is computed
over the policy struct, so the new field must be absent from the serialized policy
when unset. Evidence stays confined to the project and bounded in size.

### Non-goals

Publishing, tagging or provider workflows for any project; deriving a version from
Git tags or commits; bumping the version on the user's behalf; changing fragment
rules; adopting this in Harne8, which has its own spec after the engine release.

## 2. Requirements

### Functional

- R1: When the release policy declares `version_source`, `pose release plan`,
  `prepare` and `check` shall take the authoritative version from that file and
  reject a `--version` that differs from it.
- R2: `version_source` shall support a plain text file holding one version and a
  JSON file read through a top-level string key. A leading `v` is optional in the
  file; the compared value is always `vX.Y.Z`, and a `-dev` suffix is trimmed the
  way the engine trims its own.
- R3: The manifest's `version_evidence` shall record the declared project path, the
  kind and the compared value, never an absolute path or the file's other content.
- R4: A path outside the project, a symlink that leaves it, a missing or oversized
  file, a value that is not a semantic version and a JSON key that is absent or not
  a string shall each fail with a message naming the policy field and the cause,
  before any file is written.
- R5: When `version_source` is absent, a project that contains the engine's own
  version file shall behave exactly as today, and any other project shall fail with
  a message saying the engine version is authoritative only inside the engine
  repository and naming `version_source`.
- R6: The serialized release policy shall be unchanged when `version_source` is
  unset, so the policy digest of every existing manifest, including v6.2.0's,
  stays the same and `pose release check --version v6.2.0 --strict` still passes
  on this repository.

### Non-functional

Reading the source shall use at most one bounded file read. The existing release
tests shall keep passing unmodified except where they assumed the engine version
for a non-engine fixture, and each such test shall be listed in the execution log.

### Security

The path is project-relative and resolved through the same confinement helper the
release evidence importer uses. The file is read, never executed or interpolated,
and its content is validated against the version grammar before use or display.

### Compatibility

`release.json` keeps `schema_version: 1`; the field is additive and ignored by
engines that do not know it, which then fail with the same version mismatch as
today rather than guess. Machine output of `plan`, `prepare`, `check`, `status` and
`open-next` keeps its shape; `version_evidence.source` takes the declared path.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose/release_lifecycle.go` for the policy type, loader and a
source reader; `pose-mcp/internal/cli/release_lifecycle.go` for the evidence
selection in `releaseInputs`. The shipped policy template and the release manuals.

### Artifacts

- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/release_lifecycle_test.go
- created: pose-mcp/internal/pose/release_version_source.go
- created: pose-mcp/internal/pose/release_version_source_test.go
- modified: pose-mcp/internal/cli/release_surface_coverage_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- created: .pose/specs/2026-10-02-pose-release-version-source.md
- created: .pose/adr/2026-10-02-release-version-source-is-declared-by-the-project.md
- created: .pose/changelogs/unreleased/pose-release-version-source.md

### Delivery targets

- surface:release-version-source module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

The surface is `pose release plan|prepare|check`. Reachability and integration
evidence come from tests that drive those commands through the production
entrypoint against fixture repositories.

### API/contract changes

New optional policy object `version_source` with `path`, `kind` (`text` or `json`)
and, for `json`, `key`. No new flag or command. Error messages for the
non-engine, no-source case change from a version mismatch to an explanation.

### Data/storage changes

None. Manifests written for projects with a source carry the declared path in
`version_evidence`; manifests already in this repository are not rewritten.

### Technical risks

A fixture or test that implicitly relied on the engine version for a non-engine
repository will now fail by design, and the failures are the evidence for R5.
Reading the same project file the user edits for other purposes, such as a
`package.json` with a `version` key, is accepted by R2 and bounded by R4.

## 4. Tasks

### Planning

- [x] Confirm the reproduction on the working tree build, not only the installed
      binary.
- [x] Fix the exact policy shape and error texts.

### Implementation

- [x] Add the policy field with omitempty and the bounded reader.
- [x] Select evidence in `releaseInputs` and record the declared path.
- [x] Update the manuals in both locales and the scaffold mirrors.
- [x] Register the integration and reachability checks in the matrix.

### Validation

- [x] Text and JSON sources, the rejection cases and the no-source cases.
- [x] Manifest and policy digest of v6.2.0 unchanged; `release check` still passes.
- [ ] Full canonical matrix.

## 5. Decisions

### Decision 1

- Date: 2026-10-02
- Context: the engine version is correct evidence only in the engine repository.
  Everywhere else it is a different project's number.
- Options considered: read the version from the latest Git tag, which turns the
  tag into both input and output of the cut; take `--version` on trust, which
  removes the gate; accept an explicit project file, which keeps the existing
  "authoritative evidence" model and the project's own review of that file.
- Decision: a declared `version_source` file, with the legacy engine source kept for
  the engine repository only.
- Rationale: the workflow already says to update and review the authoritative
  version before planning; this makes that file the project's, not the binary's.
- Consequences: a project that adopts release closeout needs a versioned file;
  the policy gains one optional field.
- Falsifier: a project whose version has no file, such as one released from tags
  alone, cannot use this and would need a separate source kind.

## 6. Validation

### Strategy

Drive the commands through the production entrypoint against throwaway
repositories with an adopted policy and one fragment. Cover a text file, a JSON
file, a `v`-prefixed and a bare value, a `-dev` suffix, and every rejection in R4.
Run the engine repository's own release checks to prove R5 and R6.

### Deterministic checks

- Command: go -C pose-mcp test ./internal/pose ./internal/cli -run 'ReleaseVersionSource|ReleasePrepare|ReleasePlan|ReleaseCheck' -count=1
- Scope: policy parsing, evidence selection and the release commands
- Expected: all positive, negative and legacy cases pass

### Execution log

Reproduced before the change with the installed binary: a fixture project with an
adopted policy and a valid fragment answered `plan --version v0.2.0` with `target
v0.2.0 differs from authoritative version evidence v6.1.0`.

The package and CLI suites passed. Three of the five new CLI tests failed with the
source read disabled and passed with it restored; the two that exercise the legacy
path pass either way, as they should. Eleven existing release tests used the engine
version for a temporary non-engine directory; each now marks its fixture as the
engine repository with `markAsEngineRepository`: the five in `release_lifecycle_test.go`
(`TestReleasePrepareConsumesOnlyPendingSnapshotAndIsIdempotent`,
`TestReleasePrepareLeavesEverySpecByteIdentical`, `TestAReleasedSpecStillPassesArtifactCheckAfterTheCut`,
`TestASpecAnEarlierReleaseRewroteStillPasses`, `TestReleasePrepareFindsADatePrefixedSpec`)
and the six that share the fixtures in `release_surface_coverage_test.go`.

### Requirement trace

- R1 [satisfied] surface:release-version-source evidence:integration check:release-version-source-integration — TestReleaseVersionSourceCutsAProjectThatIsNotTheEngine plans, refuses a version that differs from the declared file, prepares and checks a project through Main.
- R2 [satisfied] surface:release-version-source evidence:integration check:release-version-source-integration — TestReleaseVersionSourceReadsTextAndJSON and TestReleaseVersionSourceReadsAJSONKey cover a bare value, a v-prefixed value, a -dev suffix and a JSON key.
- R3 [satisfied] surface:release-version-source evidence:integration check:release-version-source-integration — the cut test asserts version_evidence is exactly source, kind and value and that the manifest holds no absolute path.
- R4 [satisfied] surface:release-version-source evidence:integration check:release-version-source-integration — TestReleaseVersionSourceRejectsUnsafeOrMalformedSources covers fifteen rejections without leaking a path, and TestReleaseVersionSourceRejectionWritesNothing asserts that nothing is written.
- R5 [satisfied] surface:release-version-source evidence:integration check:release-version-source-reachability — TestReleaseVersionSourceIsRequiredOutsideTheEngineRepository and TestReleaseVersionSourceLeavesTheEngineRepositoryOnItsCompiledVersion drive Main for a project with no source and for the engine repository.
- R6 [satisfied] surface:release-version-source evidence:integration check:release-version-source-integration — TestReleasePolicyOmitsAnUnsetVersionSource; release check of v6.1.0 and v6.2.0 passed with the candidate binary.
### Known gaps

Projects released from tags alone have no source kind here.

## 7. Final Report

### Delivered scope
A project can declare the file that holds its own release version in `.pose/policy/release.json`, and `pose release plan`, `prepare` and `check` compare a cut with it. Outside the engine repository a policy without it is refused by name instead of being compared with the engine version. Policies that do not set it digest as before; v6.1.0 and v6.2.0 still verify.
### Follow-ups
No follow-ups introduced. Projects released from tags alone have no source kind; that is recorded as a known gap, not a commitment.
