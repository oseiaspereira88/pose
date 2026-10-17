---
slug: pose-test-scripts-remove-their-temp-files
status: in-progress
created_at: 2026-10-17
depends_on:
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
delivers: capability:test-scripts-remove-their-temp-files
---

# Spec: Test scripts remove the temporary files they create

## 1. Intent

### Goal

Make every shell test script of the repository remove, on exit, the temporary files it creates, and fail a check when a script that calls `mktemp` does not.

### Business value

Three scripts that CI and local validation run (`tests/quickstart/first-governed-loop.sh`, `tests/journeys/install-and-upgrade.sh`, `tests/import/migration-guides.sh`) create directories with `mktemp -d` and never remove them. One of them is the 42 MB `pose` binary they build. On a maintainer's machine, where `/tmp` is a RAM-backed tmpfs, 275 such directories (about 7 GB) accumulated within hours of consecutive validations and review runs, filled the quota and, with the memory they took, contributed to background reviews being stopped for lack of memory. The system's age-based cleanup (10 days) cannot keep up with that rate.

### Constraints

No change to what any script asserts. Scripts keep running unchanged on CI runners and under a caller's own `TMPDIR`. The helper is plain bash with no dependency.

### Non-goals

Cleaning temporary directories that other tools leave (Go caches, editors), or changing how the validation matrix runs checks.

## 2. Requirements

### Functional

- R1: A script that creates temporary files shall remove all of them when it exits, whether it succeeds, fails an assertion, errors under `set -e`, or is stopped by INT or TERM, including the files of the commands it runs.
- R2: `tests/quickstart/first-governed-loop.sh`, `tests/journeys/install-and-upgrade.sh` and `tests/import/migration-guides.sh` shall use that mechanism, and a run of each with a `TMPDIR` of its own shall leave it empty.
- R3: A check shall fail when a script under `tests/`, `scripts/`, `examples/` or `install.sh` calls `mktemp` without removing what it creates, and shall be able to fail: it shall be shown failing against the scripts as they were.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-17-pose-test-scripts-remove-their-temp-files.md
- created: .pose/starts/pose-test-scripts-remove-their-temp-files.json
- created: tests/lib/tempdir.sh
- created: tests/hygiene/temp-cleanup.sh
- modified: tests/quickstart/first-governed-loop.sh
- modified: tests/journeys/install-and-upgrade.sh
- modified: tests/import/migration-guides.sh
- modified: .github/workflows/ci.yml
- modified: scripts/verify.sh
- modified: .pose/indexes/validation-matrix.json
- created: .pose/results/test-scripts-temp-cleanup-validation.json

### Delivery targets

- capability:test-scripts-remove-their-temp-files module:pose-mcp profile:composed-capability entrypoint:tests/lib/tempdir.sh

### Approach

`tests/lib/tempdir.sh` is sourced by a script before its first `mktemp`. `pose_test_tmp_init` creates one directory, exports it as `TMPDIR` so every later `mktemp`, `go build` and `go test` of the script lands inside it, and installs an EXIT trap that removes it (restoring write permission first, for read-only Go test fixtures) plus INT and TERM traps that exit so the EXIT trap runs. Removal first restores owner access to everything under the directory (`chmod -R u+rwx`), so read-only and `0000` entries do not defeat it. The three scripts source it; the trailing `rm -rf` of the quickstart script goes, as the trap covers it.

`tests/hygiene/temp-cleanup.sh` checks, statically and on code only, that every tracked script calling `mktemp` has one of two plain forms, and rejects the rest. It drops comments and heredoc bodies (found outside quotes and arithmetic, inside `$(...)` too), follows `if`/`else` branches, loops, `case` arms, functions and groups, and cuts the file at a top-level `exit` or `return`. The helper form: `tempdir.sh` sourced once and `pose_test_tmp_init` called once, both at top level, before the first `mktemp`, no `pose_test_tmp_*` redefinition, no `TMPDIR` or `POSE_TEST_TMP` assignment, no `mktemp` with a path or `-p`, and no later trap other than a top-level `trap 'pose_test_tmp_remove' EXIT`. The own-trap form: every `mktemp` assigned whole to a variable, outside loops, functions, groups and `case` arms, that is assigned exactly once; and exactly one trap, on EXIT, in the same `if` branch as those assignments, whose command (or one-line top-level function, defined once) is only `rm -rf "$var"` for each of them. `builtin`, `unset`, `alias`, `enable`, `exec` and `command trap` are rejected, as is nesting the scan cannot follow. 57 leaking shapes (the reviewer's reproduced cases among them) and two tracked scripts changed to leak (a repeated and a looped `mktemp`) are shown rejected, and 5 correct shapes accepted. The scan is a regression guard for plain shell, not a proof for arbitrary shell: `eval` and sourced files can still hide a trap, which review of a new script covers; dynamically, that the helper leaves a sandbox `TMPDIR` empty after success, a failed assertion, an error under `set -e`, an unset variable, TERM and INT (delivered to a probe started with default signal dispositions, whose exit status 130 or 143 proves the signal ended it), including read-only and `0000` entries, each case proving by a marker that its setup ran, and shown failing against the helper with the previous `chmod -R u+w`; that a script without the helper does leave its directory (so the checks can fail); and that the quickstart script, run with a sandbox `TMPDIR`, leaves it empty. It runs in CI and as a required integration check of the matrix.

## 4. Tasks

- [x] Reproduce: 275 `/tmp/tmp.*` directories, 7 GB, from three scripts without a trap.
- [x] Helper, three scripts, hygiene check; the check fails on the previous scripts and the quickstart leaves a directory without the helper.
- [x] Run the three scripts with their own `TMPDIR`.
- [ ] Strict validation, different-actor review and closeout.

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: a trap per script that removes the directories that script created
- Selected option: one helper that makes `TMPDIR` a single directory removed on exit
- Rationale: the leak came from each script tracking its own `mktemp` results by hand (and, in one, only on the success path); redirecting `TMPDIR` also cleans what `go build`, `go test` and nested scripts create, and a new `mktemp` added later is covered without edits
- Consequences: a script that sets its own EXIT trap must call `pose_test_tmp_remove`; the static check flags a script that calls `mktemp` before the helper
- Falsifier: a script of the three that leaves anything in a `TMPDIR` of its own, or a way to end that leaves the directory

## 6. Validation

### Strategy

Show the check failing on the defect, then the three real scripts, then the module suite.

### Deterministic checks

`bash tests/hygiene/temp-cleanup.sh` fails on the previous `migration-guides.sh` (`creates temporary files with mktemp and never removes them`). The previous quickstart script, run with its own `TMPDIR`, left a `tmp.*` directory behind; the new one leaves it empty. The three scripts passed with their own `TMPDIR` and left it empty.

### Requirement trace

- R1 [satisfied] check:test — `tests/hygiene/temp-cleanup.sh` helper cases (success, assertion, error, unset variable, TERM, INT)
- R2 [satisfied] check:test — the three scripts run with their own `TMPDIR` and leave it empty; `tests/hygiene/temp-cleanup.sh` runs the quickstart
- R3 [satisfied] check:test — `tests/hygiene/temp-cleanup.sh` static scan, shown failing against the previous scripts

## 7. Final Report

### Delivered scope

Pending.

### Residual risks

Pending.

### Follow-ups

Pending.
