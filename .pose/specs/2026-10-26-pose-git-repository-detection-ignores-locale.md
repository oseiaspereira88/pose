---
slug: pose-git-repository-detection-ignores-locale
status: in-progress
created_at: 2026-10-26
completed_at:
supersedes:
depends_on:
remediates:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:git-repository-detection-ignores-locale
---

# Spec: Recognise a directory outside every repository whatever language git speaks

## 1. Intent

### Goal

Make `GitNotARepository` give the same answer under a translated git as under an English one, so the release readers keep "there is no repository" apart from "git could not answer" in every locale.

### Business value

Found on 2026-10-26 while following pose-dist `2b26a74` into Harne8: the mirror parity check, which runs the upstream suite, failed on `TestAPhaseWithAnUnreadProducerIsUnknownNotClear` and `TestLoadArchivedFragmentsReadsWhatEachManifestAttests`. Both fail on `main` itself in a `pt_BR.UTF-8` session and pass under `LC_ALL=C`, which is why CI never saw them. Bisection puts the regression at `0882b37` (B38 second review), which made `ResolveReleaseTag` depend on `GitNotARepository`, and `GitNotARepository` matches the English text `not a git repository` in git's stderr. Under a translated git, a directory outside every repository reads as a git failure: the release phase becomes unknown for projects with no releases, and an archived fragment in a project without Git is never intact. The maintainer runs POSE in `pt_BR`, so the defect reaches real use, not only the tests.

### Constraints

Only the read that interprets git's words changes; every other git command keeps the user's locale, so the messages a person reads stay in their language. The `.git` ancestor walk that guards against a lost gitdir is unchanged.

## 2. Requirements

- R1: `GitNotARepository` shall report a directory outside every repository as not a repository whatever the locale of the session, by running the probe whose message it reads in the C locale.
- R2: `ResolveReleaseTag` outside every repository shall return `ErrReleaseTagMissing` in any locale, and the release phase and archived-fragment readers that depend on it shall give in a translated session the answers they give in an English one.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-26-pose-git-repository-detection-ignores-locale.md
- created: .pose/changelogs/unreleased/pose-git-repository-detection-ignores-locale.md
- created: .pose/starts/pose-git-repository-detection-ignores-locale.json
- modified: pose-mcp/internal/pose/release_manifest_integrity.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- created: pose-mcp/internal/pose/git_locale_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:git-repository-detection-ignores-locale module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/release_lifecycle.go

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: run every git command of the release readers in the C locale
- Selected option: a separate `gitOutputUntranslated` used only by the probe whose stderr is read
- Rationale: only that probe interprets git's text; forcing the C locale everywhere would translate back every error a person reads and could change how git renders paths for no gain
- Consequences: a future reader that matches git's text must use `gitOutputUntranslated`
- Falsifier: a test that passes only under `LC_ALL=C`

## 6. Validation

### Strategy

The regression tests put a shim `git` first on the path that answers as a git outside every repository, in Portuguese unless `LC_ALL=C`, and set a `pt_BR` session; they failed before the change and pass after it, independently of the locale the test process runs in. The two upstream tests that exposed the defect run in a `pt_BR.UTF-8` session, and the whole `pose-mcp` suite passes there.

### Requirement trace

- R1 [satisfied] check:git-repository-detection-locale-integration test:TestGitNotARepositoryIgnoresTheLocale
- R2 [satisfied] capability:git-repository-detection-ignores-locale evidence:integration check:git-repository-detection-locale-integration test:TestReleaseTagOutsideARepositoryIsMissingInAnyLocale test:TestAPhaseWithAnUnreadProducerIsUnknownNotClear test:TestLoadArchivedFragmentsReadsWhatEachManifestAttests

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
