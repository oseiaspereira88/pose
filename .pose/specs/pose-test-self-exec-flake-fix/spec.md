---
slug: pose-test-self-exec-flake-fix
status: in-progress
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
delivers: capability:test-self-exec-flake-fix
changelog: none
---

# Spec: pose-test-self-exec-flake-fix

---

## 1. Intent

### Goal
Stop `pose-monorepo-validation-advisory`'s new tests from using the
currently-running `go test` binary (`os.Executable()`) as the fake check
program `pose validate` invokes as a subprocess.

### Business value
`github.com/oseiaspereira88/pose#27`: publishing v1.3.0's Release workflow
("Tests + installer E2E" job) failed `go -C pose-mcp test ./... -count=1`
with a spurious `<path>.old: no such file` error in
`TestValidateRootOnlySelectsRootModule`/`TestValidateWorkspaceResolves*`.
Reproduced independently on the local machine. Confirmed NOT reproducible
when the same tests run in isolation (`-run <name>`), and confirmed the
regular CI workflow passed the identical commit minutes before the Release
workflow's own test step failed on the very next commit (a version-only
change) — ruling out both "just these three tests are broken" and "purely
a local-machine quirk."

Root cause: the affected tests used `os.Executable()` — the path of the
live `go test` process itself — as a stand-in npm/cargo program. Under a
whole-module `go test ./...` run, the running test binary can apparently
be renamed (`.old` suffix) out from under itself mid-run, and the
subprocess self-exec then fails on a file that briefly doesn't exist at
the expected path — unrelated to the actual `--root-only`/`--workspace`
functionality under test, which was independently verified correct via
real end-to-end reproduction with the actual built binary during
`pose-monorepo-validation-advisory`'s own closeout.

### Constraints
- Fix only the three tests this session introduced
(`pose-monorepo-validation-advisory`'s `workspace_alias_test.go`). The
  pre-existing `TestValidateNativeRunsStructuredChecksWithoutShell`
  (`cli_test.go`) uses the identical risky pattern but did not reproduce
  the failure this time; hardening it is a separate follow-up, not part of
  this release-blocking fix.

### Non-goals
- Auditing every subprocess-spawning test in the codebase for the same
  risk pattern.

---

## 2. Requirements

### Functional
- R1: the three affected tests shall use a stable, standalone executable
  (never the currently-running test binary) as the fake check program.
- R2: `go -C pose-mcp test ./...` shall pass cleanly across multiple
  consecutive whole-module runs (the exact failure mode's reproduction
  shape), not merely in isolation.

### Compatibility
- No behavior change to the actual `--root-only`/`--workspace` feature —
  test-only fix.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/workspace_alias_test.go`

### Artifacts
- modified: pose-mcp/internal/cli/workspace_alias_test.go

### Delivery targets
- capability:test-self-exec-flake-fix module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: test-only change, replacing one fake-program resolution with
  another; no production code touched.

---

## 4. Tasks

### Planning
- [x] Reproduced the failure in real release CI (issue #27) and locally;
      confirmed isolation-run and CI-timing evidence rule out a pure
      per-test logic bug or pure local-machine artifact
- [x] Root-caused to `os.Executable()` self-reference under whole-module
      `go test ./...`

### Implementation
- [x] Replaced `os.Executable()` with `exec.LookPath("true")` in
      `workspaceAliasFixture` (R1)

### Validation
- [x] `go -C pose-mcp test ./...` run 3 consecutive times, `-count=1`
      each: all clean (previously failed reproducibly, both locally and
      in real CI) (R2)
- [x] `go vet ./...`, `gofmt -l .`: clean

---

## 6. Validation

### Strategy
Repeated whole-module `go test ./...` runs — the exact shape that
reproduced the original failure — rather than isolated `-run` filtering,
which never reproduced it even before the fix.

### Requirement trace
- R1 [satisfied] `workspace_alias_test.go` no longer imports `os`;
  uses `exec.LookPath("true")`.
- R2 [satisfied] 3 consecutive clean `go -C pose-mcp test ./... -count=1`
  runs after the fix.

### Known gaps
- `TestValidateNativeRunsStructuredChecksWithoutShell`'s identical
  self-exec pattern remains untouched (Constraints) — tracked as a
  follow-up.

---

## 7. Final Report

### Delivered scope
Fixed the three tests whose `os.Executable()` self-reference caused a
real, reproducible (both locally and in actual GitHub Actions release CI)
flake blocking v1.3.0's publication. No production code changed.

### Files and modules changed
- `pose-mcp/internal/cli/workspace_alias_test.go`: fake program resolution.

### Validation executed
- `go -C pose-mcp test ./...` ×3: SUCCESS each time.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l .`: clean.

### Residual risks
- The pre-existing `os.Executable()` self-reference pattern elsewhere in
  the package (`cli_test.go`) carries the same latent risk, untouched here.

### Follow-ups
- [open] harden or replace `TestValidateNativeRunsStructuredChecksWithoutShell`'s
  `os.Executable()` self-reference with the same stable-binary pattern
  used here, to close the same risk class proactively rather than waiting
  for it to flake again.
