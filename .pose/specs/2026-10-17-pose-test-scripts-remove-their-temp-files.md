---
slug: pose-test-scripts-remove-their-temp-files
status: done
created_at: 2026-10-17
depends_on:
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
delivers: capability:test-scripts-remove-their-temp-files
completed_at: 2026-10-17
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
- R3: A check shall fail when a script under `tests/`, `scripts/`, `examples/` or `install.sh` that calls `mktemp` neither uses the helper in its plain form (sourced and initialized on their own lines before the first `mktemp`) nor is pinned by the SHA-256 of the whole file after a reviewer read its own EXIT trap, so a new script cannot forget the cleanup and a pinned script cannot change without a new review. The check is a regression guard for that omission, not an analysis of arbitrary shell, and it shall be able to fail: it shall be shown failing against the scripts as they were.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-17-pose-test-scripts-remove-their-temp-files.md
- created: .pose/starts/pose-test-scripts-remove-their-temp-files.json
- created: tests/lib/tempdir.sh
- created: tests/hygiene/temp-cleanup.sh
- created: tests/hygiene/temp-cleanup.pins
- modified: tests/quickstart/first-governed-loop.sh
- modified: tests/journeys/install-and-upgrade.sh
- modified: tests/import/migration-guides.sh
- modified: examples/demo/record.sh
- modified: tests/quickstart/measure-clean-environment.sh
- modified: .github/workflows/ci.yml
- modified: scripts/verify.sh
- modified: .pose/indexes/validation-matrix.json
- created: .pose/results/test-scripts-temp-cleanup-validation.json

### Delivery targets

- capability:test-scripts-remove-their-temp-files module:pose-mcp profile:composed-capability entrypoint:tests/lib/tempdir.sh

### Approach

`tests/lib/tempdir.sh` is sourced by a script before its first `mktemp`. `pose_test_tmp_init` creates one directory (made absolute, so a relative caller `TMPDIR` keeps its meaning after a `cd`), exports it as `TMPDIR` so every later `mktemp`, `go build` and `go test` of the script lands inside it, and installs an EXIT trap that removes it (restoring write permission first, for read-only Go test fixtures) plus INT and TERM traps that exit so the EXIT trap runs. Removal first restores owner access to everything under the directory (`chmod -R u+rwx`), so read-only and `0000` entries do not defeat it. The three scripts source it; the trailing `rm -rf` of the quickstart script goes, as the trap covers it.

`tests/hygiene/temp-cleanup.sh` checks, statically, two rules over every tracked script that mentions `mktemp`. A script not in `tests/hygiene/temp-cleanup.pins` must use the helper in its plain form: `tests/lib/tempdir.sh` sourced on a line of its own, then `pose_test_tmp_init` on a line of its own, both at the start of the line and both before the first `mktemp`. The ten scripts that clean up with an EXIT trap of their own (`install.sh`, `scripts/verify.sh`, `examples/demo/record.sh` and seven test scripts) are listed in the pins file with the SHA-256 of the whole file: a change to one fails the check until a reviewer has read its allocation and trap again and run `bash tests/hygiene/temp-cleanup.sh --repin`, and a pin whose script no longer calls `mktemp` is stale. Dynamically, it checks that the helper leaves a sandbox `TMPDIR` empty after success, a failed assertion, an error under `set -e`, an unset variable, TERM and INT (delivered to a probe started with default signal dispositions, whose exit status 130 or 143 proves the signal ended it), including read-only and `0000` entries, each case proving by a marker that its setup ran, and shown failing against the helper with the previous `chmod -R u+w`; that a script without the helper does leave its directory (so the checks can fail); and that the quickstart script, run with a sandbox `TMPDIR`, leaves it empty. It runs in CI and as a required integration check of the matrix.

What the static rules do not claim: they read lines, not shell. A new script that follows the plain form and then defeats the helper with unusual syntax (a word split by a continuation, `declare -i TMPDIR`, a conditional init written across lines) passes them; the helper's own behaviour is established by the dynamic cases and by review of each new script. Eleven rounds of independent review showed that deciding cleanup from the text of arbitrary shell does not converge, so the check was reduced to the omission it can decide: forgetting the helper, and changing a pinned script unreviewed.

## 4. Tasks

- [x] Reproduce: 275 `/tmp/tmp.*` directories, 7 GB, from three scripts without a trap.
- [x] Helper, three scripts, hygiene check; the check fails on the previous scripts and the quickstart leaves a directory without the helper.
- [x] Run the three scripts with their own `TMPDIR`.
- [x] Strict validation, different-actor review and closeout.

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: a trap per script that removes the directories that script created
- Selected option: one helper that makes `TMPDIR` a single directory removed on exit
- Rationale: the leak came from each script tracking its own `mktemp` results by hand (and, in one, only on the success path); redirecting `TMPDIR` also cleans what `go build`, `go test` and nested scripts create, and a new `mktemp` added later is covered without edits
- Consequences: a script that sets its own EXIT trap must call `pose_test_tmp_remove`; a script outside the pins that does not use the plain helper form fails the check; the static check flags a script that calls `mktemp` before the helper
- Falsifier: a script of the three that leaves anything in a `TMPDIR` of its own, or a way to end that leaves the directory

## 6. Validation

### Strategy

Show the check failing on the defect, then the three real scripts, then the module suite.

### Deterministic checks

Independent review round 13 on `534653c6` passed global strict validation (137/137), explicit `pose-mcp` strict validation (134/134), `go vet ./...`, `go test ./... -count=1`, ShellCheck at warning severity, and `TestLocalVerifyCoversCurrentCIGates`. The registered `test-scripts-remove-their-temp-files-integration` producer executes `tests/hygiene/temp-cleanup.sh`.

Independent helper probes reached setup markers, removed restricted 0500/0400/0000 descendants, and cleaned up on success, assertion failure, set-e error, unset variable, INT (130) and TERM (143). Old-permissions and missing-signal-handler controls make the gate fail. The three real scripts leave their caller TMPDIR empty with absolute paths, relative paths, CDPATH unset or exported as dot, and the reviewed binary/previous options.

The revised R3 gate rejects original scripts, seven invalid helper forms in a newly tracked script, changes to each of the ten pinned files, deleted/no-longer-allocating pinned scripts and a nonexistent pin. A valid helper form and an explicitly reviewed re-pin pass. All review mutations were restored. The original quickstart without the helper leaves a directory, proving the empty-TMPDIR check can fail.

The demo succeeds and cleans up after a failed build. The actual clean-environment Docker measurement executes the installed pose 7.4.1 and staged quickstart/helper successfully (12.486 seconds for the measured command path), leaving its host TMPDIR empty. A Docker removal failure no longer skips host cleanup; the previous measurement script leaks under the same injected failure. Compact results and hashes of full external outputs are recorded in [independent review evidence](../results/test-scripts-temp-cleanup-independent-review.json).

### Requirement trace

The named producers establish behavior; strict lint checks structure and does not establish requirement satisfaction.

- R1 [satisfied] check:test-scripts-remove-their-temp-files-integration — `tests/hygiene/temp-cleanup.sh` dynamically exercises the helper; independent permission/failure/signal/CDPATH probes and the old/new Docker-cleanup control extend its negative evidence.
- R2 [satisfied] check:quickstart-real-lifecycle-integration check:install-and-upgrade-journeys-integration — real producers `tests/quickstart/first-governed-loop.sh`, `tests/import/migration-guides.sh` and `tests/journeys/install-and-upgrade.sh` were also run independently with their own TMPDIR. All leave it empty, including binary/previous/cwd and relative-TMPDIR variants; the measurement producer proves the staged helper dependency.
- R3 [satisfied] check:test-scripts-remove-their-temp-files-integration — the helper-form/pin gate plus independent tracked-script and pin mutations reject the omission/change/staleness cases, and original scripts fail. This is the maintainer-authorized revised R3, not an arbitrary-shell cleanup proof.

## 7. Final Report

### Delivered scope

The helper owns an absolute temporary root, exports TMPDIR and removes restricted descendants on the exits and signals required by R1. The three scripts adopt it without changing their assertions. CI, local verification and the validation matrix execute the hygiene gate. The revised R3 enforces the plain helper form or a reviewed whole-file SHA-256 pin; the demo uses one root, and the clean-environment producer stages the helper and survives Docker cleanup failure.

Maintainer-authorized independent review by `agent:independent-gpt-6.1-sol-review` (Codex/OpenAI, a different vendor from the Claude/Anthropic implementer) approved round 13. Identity and separation are declared; pose-dist attestations are unsigned. Bundle `rvb-6167567bf3fa6ac4` and attestation `rva-5358c584e2a8bd4f` bind the review. Native review verification, guarded close, strict spec lint and strict check passed. See the [review report](../reports/2026-10-17-pose-test-scripts-temp-cleanup-review.md) and [compact evidence](../results/test-scripts-temp-cleanup-independent-review.json).

This hygiene delivery is outside B01–B44; no implementation-plan unit is created. The two program roadmaps record its closeout while retaining their existing lifecycle and maintainer merge authority for PR #192.

### Residual risks

SIGKILL and machine/power loss cannot execute a shell trap and may leave a root; the independently exercised cleanup guarantee covers the exits and signals in R1. Persistent Go caches and tools that ignore TMPDIR remain outside the stated cleanup scope.

R3 deliberately reads lines rather than analyzing arbitrary shell. Unusual syntax or later helper sabotage can pass the plain-form rule; each new script and re-pin requires reviewer inspection. A pin detects whole-file changes and does not authenticate who reviewed them. The `agent:independent-` prefix remains an engine-required declaration until `pose-delegated-review-run-provenance` replaces it, rather than signed session provenance.

### Follow-ups

No outstanding corrective follow-up remains for this delivery. New scripts and changed pins continue through the review obligations of R3.
