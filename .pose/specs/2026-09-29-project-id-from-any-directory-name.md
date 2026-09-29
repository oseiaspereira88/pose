---
slug: project-id-from-any-directory-name
status: done
created_at: 2026-09-29
completed_at: 2026-09-29
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
delivers: surface:project-id-derivation
---

# Spec: A project installs and indexes whatever its directory is called

## 1. Intent

### Goal

`pose install` and `pose index` succeed in a Git repository whose directory name is
not a slug — `MyApp`, `Acme Portal`, the `tmp.9Sz2E9ps7b` that `mktemp` returns. On
6.0.0 both fail with `pose index: invalid-project-configuration` and nothing else.

### Business value

Uppercase directory names are the norm on macOS and Windows and common everywhere
else, so a first-time user of 6.0.0 meets a failed install before any POSE concept.
The failure also kept `main`'s CI red from 2026-09-21: the migration-guides script
installs into `mktemp -d`. A CI that stays red stops being read, and v6.0.0 was
published from a commit whose CI was red for this and for a second, unrelated
defect (spec `release-runs-the-ci-gates`).

### Constraints

A project id that someone declared — `POSE_DEFAULT_PROJECT_ID`, `POSE_PROJECT_ROOTS`,
`--project-id` — is identity and may be cited by another repository's `xref:`. It is
never rewritten silently. Only the id POSE derives when none is declared changes, and
it changes only for names that were never valid.

### Non-goals

Relaxing the project-id grammar. Ids stay slugs, because they are parsed inside
`xref:<project>/<kind>:<slug>` references.

## 2. Requirements

- R1: A project that declares no id receives `proj.` plus its directory name folded
  into a slug: lowercased, with every character a slug cannot hold replaced by `-`.
  A name that is already a valid slug keeps the id it had.
- R2: `pose install` seeds `.mcp.json` with that id, and `pose index` succeeds, for
  directory names with uppercase letters, spaces and dots.
- R3: An invalid declared id is still refused, and the refusal names the variable,
  the value and the id to declare instead. `pose index` prints that cause instead
  of only `invalid-project-configuration`.
- R4: `pose doctor` reports an invalid id stamped in `.mcp.json` by an earlier engine
  as a fixable `mcp.config` warning, and `pose doctor --fix --yes` replaces only that
  id, preserving the rest of the server entry.
- R5: `pose install` refuses an invalid `--project-id` before writing anything, and
  a reinstall does not carry forward an invalid id recovered from `AGENTS.md` or
  `.mcp.json`; a valid declared id survives a reinstall unchanged.

## 3. Technical Plan

The id was derived as `"proj." + filepath.Base(root)` in five places — the artifact
resolver, `install`, the manual merge, `doctor`'s MCP repair and the MCP server
bootstrap — and 6.0.0 started validating it as a slug in the resolver
(`pose-qualified-artifact-resolution`). One helper, `ProjectIDFor`, now folds a name
into a slug and every derivation uses it. The existing MCP repair only migrated
legacy `pose-mcp` entries and left a native entry untouched, so it also learns to
replace an invalid id in a native entry.

Test roots come from `t.TempDir()`, whose basenames (`001`) are always slugs; that is
why no Go test saw the regression. The new tests name directories the way people do.

### Artifacts

- created: .pose/specs/2026-09-29-project-id-from-any-directory-name.md
- created: .pose/changelogs/unreleased/project-id-from-any-directory-name.md
- created: pose-mcp/internal/cli/project_id_directory_name_test.go
- modified: pose-mcp/internal/pose/artifact_ref.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/bootstrap/bootstrap.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- surface:project-id-derivation module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

No stored identity changes: a valid derived id is unchanged, and an invalid one never
resolved on 6.0.0. Reverting restores the failure.

## 4. Tasks

- [x] Reproduce the failed install in a directory named like `MyApp`, and the CI failure it caused.
- [x] Fold derived ids into slugs through one helper used by every derivation.
- [x] Name the cause when a declared id is refused, and surface it from `pose index`.
- [x] Let `pose doctor --fix` repair an invalid id stamped by an earlier engine.
- [x] Prove the new test fails with the old derivation.
- [x] Refuse an invalid `--project-id` and stop a reinstall from carrying forward an id that never resolved.

## 5. Decisions

### Decision D1

- Status: active
- Fold the derived id instead of accepting uppercase ids. The grammar is shared with
  `xref:` parsing and with ids other repositories already cite; widening it is a
  contract change for every consumer, while folding changes only ids that never
  worked.

### Decision D2

- Status: active
- Refuse, do not fold, an invalid declared id. A declared id may be referenced from
  elsewhere; rewriting it at read time would make two spellings of one project. The
  places that rewrite are explicit writes of the machinery — `pose install`/`update`
  and `doctor --fix` — and only an id no resolver accepts; a valid declared id,
  including one that differs from the directory name, survives both.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Install and index in named directories | `go test ./internal/cli -run AnyDirectoryName -count=1` | `MyApp`, `Acme Portal`, `tmp.9Sz2E9`, `my-app` all install and index |
| Declared invalid id | `go test ./internal/cli -run DeclaredProjectID -count=1` | refusal names the variable, the value and the replacement |
| Doctor repair | `go test ./internal/cli -run InvalidStampedProjectID -count=1` | fixable warning, repair, then ok |
| Install and reinstall ids | `go test ./internal/cli -run 'InvalidExplicitProjectID|NeverResolved|ValidDeclaredProjectID' -count=1` | explicit invalid id refused with exit 2 and nothing written; stale id replaced; valid declared id kept |
| Migration guides (CI step) | `bash tests/import/migration-guides.sh` | `All documented migration claims hold.` |
| Registered producer | `pose validate` check `project-id-derivation-integration` | the six tests above, as integration evidence for the surface |
| Full suite | `go test ./... -count=1` | pass |

### Execution log

2026-09-29: CI's `Documented migration paths still hold` step failed on every push
since 2026-09-21 with `pose install failed in /tmp/tmp.9Sz2E9ps7b`. Reproduced
locally: `pose install` delivered the machinery and then reported
`pose index: invalid-project-configuration`. The swallowed cause was
`invalid-project-id` for `proj.tmp.9Sz2E9ps7b`. Directories `myapp`, `my.app` and
`my_app` indexed; `MyApp` did not.

2026-09-29, implemented. With the derivation reverted to the 6.0.0 form, the new
test fails for `MyApp`, `Acme Portal` and `tmp.9Sz2E9` and passes for `my-app`; with
the fix it passes for all four. The full Go suite and the migration-guides script
pass.

2026-09-29, review. Reading the diff against the install path found two gaps in the
first commit. `--project-id` was never validated, so an operator could stamp an id
every resolver refuses. And a reinstall recovers the declared id from `AGENTS.md` and
`.mcp.json` before deriving one, so an instance that 5.x stamped as `proj.MyApp` kept
it through `pose update --force`; only `doctor --fix` repaired it. Both are closed
with tests; with the recovery check removed, the reinstall test fails with
`reinstall kept "proj.MyApp"`.

### Closeout

2026-09-29 UTC, measured in a clone holding only this branch, because `origin/main`
reaches the squash of PR #119, which carries this spec's trailer beside three
others and inflates its change set. Full matrix 38/38 into the results path; bundle
`rvb-81b7d02883a70ebb`, 35 evidence items; attestation `rva-90c76469987ce75a`,
`agent:claude-opus-5-5`, approved with five explicit judgments; `review-check`
fresh and approved.

### Requirement trace

- R1 [satisfied] surface:project-id-derivation evidence:integration check:project-id-derivation-integration test:TestInstallAndIndexAcceptAnyDirectoryName — MyApp,
  Acme Portal and tmp.9Sz2E9 derive folded ids; my-app keeps proj.my-app
- R2 [satisfied] surface:project-id-derivation evidence:integration check:project-id-derivation-integration test:TestInstallAndIndexAcceptAnyDirectoryName — install
  seeds .mcp.json with the folded id and pose index exits 0 for every name
- R3 [satisfied] surface:project-id-derivation evidence:integration check:project-id-derivation-integration test:TestIndexNamesWhyADeclaredProjectIDIsRefused — a declared
  proj.MyApp is refused, naming POSE_DEFAULT_PROJECT_ID, the value and proj.myapp
- R4 [satisfied] surface:project-id-derivation evidence:integration check:project-id-derivation-integration test:TestDoctorRepairsAnInvalidStampedProjectID — doctor
  reports a fixable mcp.config warning, the repair declares proj.myapp, then ok
- R5 [satisfied] surface:project-id-derivation evidence:integration check:project-id-derivation-integration test:TestInstallRefusesAnInvalidExplicitProjectID
  test:TestReinstallReplacesAnIDThatNeverResolved
  test:TestReinstallKeepsAValidDeclaredProjectID — an invalid --project-id exits 2
  before writing; a reinstall replaces a stale invalid id and keeps a valid one

## 7. Final Report

### Scope delivered

A repository installs and indexes whatever its directory is called. Derived ids are
folded into slugs by one helper; declared ids are never rewritten at read time, and
the one that earlier engines stamped as `proj.MyApp` is repaired by `doctor --fix` or
the next reinstall.

### Residual risks

Two directories whose names fold to the same slug (`MyApp`, `myapp`) derive the same
id. They are separate repositories, and each resolves only its own root unless an
operator registers both in `POSE_PROJECT_ROOTS`, where the existing
`conflicting-project-binding` refusal applies.

### Follow-ups

None.
