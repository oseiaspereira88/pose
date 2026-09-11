---
slug: pose-mcp-server-survives-self-update
status: in-progress
completed_at:
created_at: 2026-09-10
supersedes:
depends_on: pose-self-update-handoff-path
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: A running MCP server keeps working after `pose update`

## 1. Intent

### Goal
A `pose serve-mcp` started before `pose update` replaced its binary shall still
be able to run the CLI-backed tools it serves.

### Business value
Most MCP tools answer by running the pose CLI, and the server found that CLI by
calling `os.Executable()` on every call. `pose update` renames the running binary
to `.old`, writes the new one in its place and removes the `.old`. On Linux
`os.Executable()` resolves through `/proc/self/exe`, so from that moment it named
the removed file, and every CLI-backed tool failed with
`fork/exec …/pose.old: no such file or directory` until the server was
restarted.

It was found by using it: `pose_get_followups` failed that way in this
repository's own session, served by a process started two days and one update
earlier. Nothing in the failure says the server needs a restart — it reads as a
missing binary.

This is the defect v2.0.0 hit in the update handoff, and the fix there was
local to the handoff: `performSelfUpdate` returns the path it wrote rather than
resolving one afterwards. The one caller that outlives an update kept resolving.

### Constraints
- `POSE_EXECUTABLE` keeps precedence: an operator who configured a path gets
  that path.

### Non-goals
- Keeping a server on the engine it started with. After an update the path holds
  the new engine, and a freshly started server would run that one too.

---

## 2. Requirements

### Functional
- R1: A CLI-backed tool shall run after `pose update` replaced the binary the
  server was started from.
- R2: It shall run the binary installed at that path, not a name the update
  removed.

### Non-functional
- The test runs the real sequence: a server started from a built binary, the
  real `pose update` against a local release server, then a tool call.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/cli.go` — how the server finds the CLI it runs

### Artifacts
- created: .pose/specs/2026-09-10-pose-mcp-server-survives-self-update.md
- renamed: .pose/changelogs/unreleased/pose-mcp-server-survives-self-update.md -> .pose/changelogs/v5.0.1/pose-mcp-server-survives-self-update.md
- created: pose-mcp/internal/cli/mcp_self_update_test.go
- modified: pose-mcp/internal/pose/cli.go
- modified: pose-mcp/internal/cli/self_update_release_test.go

### Technical risks
- A server older than the binary now at its path runs a newer CLI. Its commands
  are the ones that CLI was released with, so a flag the server passes and the
  newer CLI removed fails with that CLI's usage error, visibly — instead of
  every tool failing with a misleading one.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Reproduce with a running server and a real update (R1)
- [x] Increment 2: Resolve the executable once, when the process starts (R1, R2)

### Validation
- [x] The test fails without the fix, with the error observed in use

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: a server outliving an update can run the binary now at its start
  path, or the image it is itself running — on Linux `/proc/self/exe` still
  executes a deleted file.
- Decision: the start path, resolved once when the process starts.
- Rationale: it behaves the same on every platform, while executing the running
  image is Linux-only; macOS already reports the launch path, so it already runs
  the new binary. It is also what a restarted server does, so restarting stops
  being a way to change the answer, only a way to pick up the new server.

### Decision 2
- Date: 2026-09-10
- Context: the self-update test and the new one both need a local release server
  and a pose binary built against it.
- Decision: extract both into helpers the two tests share.
- Rationale: the new test depends on running the real `pose update`, not a copy
  of its rename sequence — a copied sequence would pass whatever the update does.

---

## 6. Validation

### Strategy
Reproduce the failure as observed, then require the tool to run the updated
binary.

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
- Date: 2026-09-10
- Environment: local, Go 1.26, Linux
- Notes: before the fix, `TestMCPServerRunsTheCLIAfterTheBinaryIsUpdated`
  failed with `pose check: fork/exec …/install/pose.old: no such file or
  directory` — the message this repository's session reported. After it, the
  tool's output is the fixture binary's, carrying `args=[check`. Passed three
  consecutive runs; the full `pose-mcp` suite passes.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestMCPServerRunsTheCLIAfterTheBinaryIsUpdated starts `pose serve-mcp --stdio`, runs the real `pose update` on the binary it was started from, then calls pose_check and requires no `.old` in the answer>
- R2 [satisfied] <the same test requires the tool's output to be the updated binary's own, `FIXTURE-POSE-3.0.0 args=[check`>

### Known gaps
- Windows is skipped, as it is for the self-update test: the rename there is a
  different path.

---

## 7. Final Report

### Summary
A server outliving an update runs the engine now installed where it started,
instead of failing every CLI-backed tool with a missing-binary error.

### Follow-ups

- [open] `pose serve-mcp --stdio` ignores SIGTERM until its next request: `bootstrap.Run` catches the signal with `signal.NotifyContext`, but `ServeStdio` blocks in `scanner.Scan()` and checks the context only after a line arrives, then exits without answering it. Observed stopping this repository's own server after installing 5.0.1: SIGTERM left it running, and the next tool call closed the connection. Make the stdio loop return when the context is cancelled (owner:unowned crit:medium review:2026-10-10)
