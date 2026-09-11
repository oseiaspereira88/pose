---
slug: pose-stdio-server-honours-sigterm
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-mcp-server-survives-self-update
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: A stdio MCP server stops when it is asked to

## 1. Intent

### Goal
`pose serve-mcp --stdio` shall exit when it receives SIGTERM or an interrupt,
whether or not a request is pending.

### Business value
The server catches SIGTERM through `signal.NotifyContext`, so the default
action — terminate — never runs. Its loop then blocks reading stdin and looked at
the context only after the next line arrived. SIGTERM therefore changed nothing
until a client sent another request, which the server then dropped: it returned
without answering and the client saw the connection close.

Found while restarting this repository's own MCP server to pick up 5.0.1: the
process survived SIGTERM, and the next tool call came back as "Connection
closed". A client or supervisor that stops servers with SIGTERM — the usual
way — leaves them running, one per session, until it escalates to SIGKILL or
never does.

### Constraints
- A request already being handled is finished before the loop looks again.

### Non-goals
- Draining in-flight tool calls on shutdown; stdio handles one at a time.

---

## 2. Requirements

### Functional
- R1: The stdio server shall exit within a bounded time after SIGTERM while idle
  with stdin open.
- R2: A requested shutdown shall exit with status 0.
- R3: The `mcp-agent-interop` capability, stale since 2026-08-21, shall be
  reassessed with this fix and `pose-mcp-server-survives-self-update` as evidence.

### Non-functional
- EOF on stdin still ends the server, and a read error is still returned.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/mcpserver/server.go` — `ServeStdio`

### Artifacts
- created: .pose/specs/2026-09-11-pose-stdio-server-honours-sigterm.md
- renamed: .pose/changelogs/unreleased/pose-stdio-server-honours-sigterm.md -> .pose/changelogs/v5.0.2/pose-stdio-server-honours-sigterm.md
- created: pose-mcp/internal/cli/mcp_sigterm_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: .pose/specs/2026-09-10-pose-mcp-server-survives-self-update.md
- modified: .pose/capabilities/assessment.md
- modified: .pose/capabilities/history.jsonl

### Technical risks
- The reader goroutine stays blocked on stdin after a shutdown; the process is
  exiting, so it is not leaked in any meaningful sense.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Read stdin apart from the loop, and let the loop select on the context (R1, R2)
- [x] Increment 2: Reassess `mcp-agent-interop` and snapshot (R3)

### Validation
- [x] The test fails against the previous loop with the server still running

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: a cancelled context used to surface as `context canceled`, which the
  bootstrap turns into `log.Fatal` and exit 1 — when the loop ever reached it.
- Decision: return nil on a requested shutdown.
- Rationale: stopping on SIGTERM is the server doing what it was asked; an exit
  status of 1 would make every clean stop look like a crash to a supervisor.

---

## 6. Validation

### Strategy
Start a real server, confirm it is idle in its read loop, send SIGTERM, and time
its exit.

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
- Environment: local, Go 1.26, Linux
- Notes: against the previous loop, TestStdioMCPServerExitsOnSIGTERMWhileIdle
  fails with the server still running 10 s after SIGTERM. With the fix it exits
  with status 0 at once; passed three consecutive runs, and the self-update test
  from `pose-mcp-server-survives-self-update` still passes.

### Results summary
- Successes: R1, R2 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestStdioMCPServerExitsOnSIGTERMWhileIdle answers a ping, sends SIGTERM with stdin open, and requires the process to exit within 10 s>
- R2 [satisfied] <the same test requires server.Wait() to return no error>
- R3 [satisfied] <assessment.md cites both lifecycle fixes and the triggering spec, keeps the score at 5 with the reason; `pose assess snapshot` cleared the mark and appended a snapshot>

### Known gaps
- Windows is skipped; SIGTERM is POSIX. Ctrl+C there takes the same path through
  the context but is not exercised.

---

## 7. Final Report

### Summary
Stopping a stdio MCP server now stops it.

### Follow-ups
