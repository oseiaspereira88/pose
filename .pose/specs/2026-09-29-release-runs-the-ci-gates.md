---
slug: release-runs-the-ci-gates
status: in-progress
created_at: 2026-09-29
completed_at:
supersedes:
depends_on:
priority: 0
components: ci, docs
task_type: bugfix
delivers: governance:release-requires-green-ci
---

# Spec: A release runs, and passes, the gates CI runs

## 1. Intent

### Goal

The release workflow cannot publish from a commit whose CI gates fail. The
Portuguese README states the released version, and the public claims gate passes.

### Business value

v6.0.0 was published with a Portuguese README whose checksum-pinned install
instructions download 5.0.8. The public claims gate reports exactly that, but it
runs only in `ci.yml`, and `release.yml` runs its own subset of CI's checks without
it. CI on `main` had been red since 2026-09-21 for another defect (spec
`project-id-from-any-directory-name`), so the new failure arrived at a job nobody was
reading. This is the second time: `ci.yml` itself records that CI was red on `main`
from 2026-08-22 for a gate defect nobody investigated. A release gate that consults
only part of CI lets a red CI become normal and lets real defects ship under it.

### Constraints

No change to the gates themselves. `release.yml` keeps its own steps, including the
cross-version test it must never skip. The tag and assets of v6.0.0 stay immutable:
its notes and README are not rewritten; the fix reaches users through the next
release.

### Non-goals

Requiring CI on a different commit (a PR, `main`) than the one tagged. The workflow
runs CI at the commit it publishes, so the verdict cannot refer to another tree.

## 2. Requirements

- R1: `release.yml` runs the whole of `ci.yml` at the commit it publishes, and the
  release job starts only when that run passes.
- R2: `ci.yml` stays one definition: the release calls it, it is not copied.
- R3: `README.pt-BR.md` pins the released version in both install snippets, and
  `pose public-claims --strict` passes.
- R4: A contract test fails when `ci.yml` stops being callable, when the `ci`
  job stops calling it, or when `release` stops needing `ci`.

## 3. Technical Plan

Add `workflow_call` to `ci.yml`. In `release.yml`, add a `ci` job that calls it with
read-only permissions and make `release` depend on it. The Portuguese snippets take
the version the English ones already carry.

### Artifacts

- created: .pose/specs/2026-09-29-release-runs-the-ci-gates.md
- created: .pose/changelogs/unreleased/release-runs-the-ci-gates.md
- modified: .github/workflows/ci.yml
- modified: .github/workflows/release.yml
- modified: README.pt-BR.md
- created: pose-mcp/internal/version/release_needs_ci_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:release-requires-green-ci module:. profile:release-governance entrypoint:.github/workflows/release.yml

### Rollout and reversal

The next tag push runs CI before the release job. A release takes as long as CI plus
the release job. Reverting is removing `needs: ci` and the `ci` job.

## 4. Tasks

- [x] Identify why v6.0.0 was published while CI failed on it.
- [x] Pin the released version in the Portuguese README.
- [x] Make CI callable and make the release depend on it.
- [x] Observe a release run in which the `ci` job gates the `release` job.

## 5. Decisions

### Decision D1

- Status: active
- Call `ci.yml` instead of copying the public claims step into `release.yml`. Copying
  one step fixes this instance and leaves the next gate added to CI outside the
  release again; the gap was the subset, not the missing step.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Public claims | `pose public-claims --strict` | `public-claims.errors=0` |
| Workflow structure | `go -C pose-mcp test ./internal/version -count=1` | release workflow contract tests pass |
| Workflow shape | YAML parse of both workflows | `ci.yml` triggers include `workflow_call`; `release` needs `ci` |
| Wiring contract | `go -C pose-mcp test ./internal/version -run 'ReleaseWorkflowWaitsForCI|ReleaseNeedsCIFindings' -count=1` | the repository's workflows pass; each removed link is one finding; a commented `needs` is not wiring |
| Release run | next tag or `workflow_dispatch` rehearsal | `ci` job runs first; `release` waits for it |

### Execution log

2026-09-29: CI's `governance` job fails at `Public claims gate` with
`README.pt-BR.md: declares version 5.0.8 but the released version is 6.0.0`, on the
commits that froze and recorded v6.0.0. The tag's `README.pt-BR.md` has `V=5.0.8` at
lines 115 and 127. Commit `6d7e501` aligned the English README and `docs/ci.md` to
the candidate and missed the Portuguese one. `release.yml` does not run the public
claims gate.

2026-09-29, implemented. `pose public-claims --strict` passes. Both workflows parse;
the release workflow contract tests pass. No `actionlint` is installed here, so the
workflow semantics are observed only by the next release run.

2026-09-29, observed on the v6.0.1 release. Run `36516495961`, triggered by the
tag push at commit `3847ad2`: `ci / test` ran 03:17:44–03:21:02 and
`ci / governance` 03:17:43–03:21:24, both successful; `release` started at
03:21:27, three seconds after the last CI job ended, and succeeded at 03:27:52.
The release job's `needs: ci` held it until CI passed. Independent verification
run `36517247097` then verified the published v6.0.1.

2026-09-29, closeout. The release run proves the wiring once; nothing stopped a
later edit from dropping `needs: ci` again. `TestReleaseWorkflowWaitsForCI` now
pins all three links; with `needs: ci` removed from `release.yml` it fails with
"release.yml's `release` job does not need `ci`".

### Requirement trace

- R1 [pending] release run with `ci` gating `release`
- R2 [pending] .github/workflows/release.yml calls .github/workflows/ci.yml
- R3 [pending] check:public-claims
- R4 [pending] test:TestReleaseWorkflowWaitsForCI

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

The release is slower by the length of CI, including the delivery images step.
Accepted: a release is rare and the cost is time, not correctness.

### Follow-ups

None.
