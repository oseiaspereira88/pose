---
slug: pose-self-update-handoff-path
status: in-progress
created_at: 2026-09-08
completed_at:
supersedes:
depends_on: pose-contract-adoption-stamp
priority: 0
components: pose-mcp
delivers:
---

# Spec: The self-update handoff runs the binary it just wrote

## 1. Intent

### Goal
Hand off to the replaced executable by the path it was written to, instead of
re-resolving it after this process has renamed itself out of the way.

### Business value
`pose-contract-adoption-stamp` made `pose update` hand off to the binary it
downloads, so a migration shipped in a release applies on the update that
delivers it. The handoff called `os.Executable()` to find that binary.

It is the wrong call at that moment. `performSelfUpdate` renames the running
executable to `<path>.old`, copies the new one into `<path>`, and removes the
backup. On Linux `os.Executable()` resolves through `/proc/self/exe`, which
follows the inode — so after the rename it returns the `.old` name, which no
longer exists. The v2.0.0 release run failed on exactly that:

```
[WARN] running the updated binary: fork/exec .../cli.test.old: no such file
```

Every local run was green, and could not have been otherwise: reaching the
handoff requires actually downloading a newer release, so an offline or
already-current run returns before it. The end-to-end check recorded in that
spec exercised the stamp, not the handoff — the claim was accurate about what it
tested and the gap was in what it did not.

### Constraints
- The fix must be testable without a network, or it is the same gap again.

### Non-goals
- Replacing this process rather than running a child. `syscall.Exec` has no
  Windows equivalent, and the child already relays output and exit code.

---

## 2. Requirements

### Functional
- R1: `performSelfUpdate` shall return the path it wrote the new binary to, and
  the handoff shall run that path.
- R2: The original arguments shall reach the handed-off process, with
  `--no-self` so it terminates.
- R3: The handoff shall propagate the child's exit code.

### Non-functional
- The handoff is covered by tests that do not require a release download.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/maintenance.go` — self-update and handoff

### Artifacts
- created: .pose/specs/2026-09-08-pose-self-update-handoff-path.md
- renamed: .pose/changelogs/unreleased/pose-self-update-handoff-path.md -> .pose/changelogs/v2.0.1/pose-self-update-handoff-path.md
- created: pose-mcp/internal/cli/self_update_handoff_test.go
- modified: pose-mcp/internal/cli/maintenance.go

### Technical risks
- The returned path doubles as the "did it replace" signal, so an empty string
  means no replacement. That is one value carrying two meanings; the alternative
  is a second return value that can disagree with the first, which is what
  produced this defect.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Return and use the written path (R1)
- [x] Increment 2: Cover the handoff without a network (R2, R3)

### Validation
- [x] The defect reproduced by a test that fails against the previous code

---

## 5. Decisions

### Decision 1
- Date: 2026-09-08
- Context: how to test a path that only runs after a real download.
- Decision: test `runSelfUpdatedBinary` directly against a stub executable.
- Rationale: the defect is entirely in which path is executed, which the
  function's own contract expresses. Testing it there needs no network and
  fails against the previous code, where the argument was ignored in favour of
  `os.Executable()`. Waiting for a release run to exercise it is what let this
  ship.

---

## 6. Validation

### Strategy
Run the handoff against a stub whose output identifies it, and confirm the
assertion fails when the path is re-resolved instead.

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
- Date: 2026-09-08
- Environment: local, Go 1.26
- Notes: all eight packages pass. Restoring `os.Executable()` in place of the
  passed path fails `TestSelfUpdateHandoffRunsThePathItWasGiven` — under `go
  test` it resolves to the test binary, which does not answer `update`, so the
  handoff returns 1. That is the same shape as the release failure, reproduced
  without a download.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <performSelfUpdate returns execPath and "" when it replaced nothing; cmdUpdate hands that path to runSelfUpdatedBinary, which no longer resolves anything>
- R2 [satisfied] <TestSelfUpdateHandoffRunsThePathItWasGiven asserts the stub receives `update --no-self --locale pt-BR`>
- R3 [satisfied] <TestSelfUpdateHandoffPropagatesTheExitCode asserts a child exiting 3 makes the update report 3, so a failure after the handoff is not reported as success>

### Known gaps
- Nothing exercises the full download-replace-handoff sequence; the release
  workflow is still the first place that runs it end to end.

---

## 7. Final Report

### Follow-ups

- [done] Exercise the download-replace-handoff sequence against a local release fixture, so the release run is not the first to run it. Done in `pose-release-boundary-rehearsal`: a local server serves the release metadata and archive, a pose binary built at an older version downloads and replaces itself, and restoring `os.Executable()` in the handoff reproduces the v2.0.0 `fork/exec .../pose.old` failure locally. — owner:unowned crit:medium review:2026-11-08
