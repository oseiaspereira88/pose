---
title: Recovery reads evidence, not presence
status: accepted
date: 2026-10-16
decision_type: architecture
---

# Recovery reads evidence, not presence

## Context

Spec `pose-journal-checkpoint-and-lock-recovery` (B13) reproduced on the
engine at `b8c33dad` how three pieces of operation state answer a question
other than the one their readers ask (front 9 CONS-02, CONS-07, CONS-12;
probes in `.pose/results/b13-front09-probes.json`):

- An action-request journal accepted a second JSON value after the event on a
  line, a duplicate field (the last value counted), an unsupported
  `schema_version` and an empty line, and read the request as if nothing were
  there. A truncated last line made the whole request unreadable, with no
  byte position, hash or way to tell an interrupted append from corruption.
- A closeout checkpoint that did not parse, had another schema or named
  another scope read as absent: `close --plan` printed exactly what it prints
  when no apply was interrupted, and an apply started over it. A failed write
  was reported, but the file was rewritten in place and its removal at the
  end of a run was never checked. A checkpoint listing a step as completed
  that the repository did not reflect was resumed anyway: the step ran again.
- The start lock was an empty `O_EXCL` file. It survived a crash, carried no
  owner and could only be removed by a person who had to guess whether a
  start was still running.

## Options considered

1. **A general journal, publisher and lock service.** Covers every writer,
   but it is B14 (publisher/CAS) and B35 (runtime) and would change
   contracts this unit does not own.
2. **Repair automatically.** Truncate interrupted tails, drop corrupt
   checkpoints and break old locks by age. Makes gates pass by deleting the
   evidence they should read; a waiver or an answer can be the lost tail,
   and a lock can be old and still held.
3. **Read each artifact by the evidence it holds, report what it is, and
   leave every repair to a person or to an explicit, verified command.**

Option 3 is chosen.

## Decision

**Journals** (`internal/pose/journal_integrity.go`). Every byte is read:
each line ends with a newline and holds exactly one JSON object, with no
duplicate field, unknown field or trailing value, the supported schema and
the next sequence number; a line is at most 1 MiB. Escaped control
characters are text and stay valid — journals in use carry them. The first
bytes that do not read are an `*ActionJournalError` (`ErrActionJournalInvalid`)
naming the request, path, line, byte offset, length and SHA-256 of those
bytes, and the events before them. An unterminated last line is an
`interrupted-tail`, even when its bytes parse; anything else is `corrupt`,
wherever it is. Only an interrupted tail gets a repair proposal: text saying
where truncating keeps the complete events, after a person confirms no writer
is running and nothing is lost. The engine never truncates, and an append to
a journal that does not read is refused. CLI (`action show|list`, both
renderers) and MCP (`error_code: action_journal_invalid`) report the location
and the next step.

**Checkpoints** (`internal/pose/closeout_plan.go`). A checkpoint is
`absent`, `valid`, `corrupt`, `unsupported` or `unavailable`; it is valid only
when it reads strictly, has schema 1, names this scope, a plan digest, an RFC
3339 start and known steps, each once. The checkpoint is a record of
progress, not an authoritative artifact — the plan is computed from the
repository — so it is replaced through an exclusive temporary file and a
rename (never torn) and is not synced: a crash may lose its last update, not
present a broken one. Its removal at the end of a run is checked and a
failure is an `unconfirmed` entry of the result. `close --plan` shows the
state (`checkpoint` in JSON, outside the plan digest). An apply or resume
over a checkpoint that is not absent or valid is `blocked` before any step,
with the next step to inspect and remove it; the engine never removes it. A
valid checkpoint that records an effectful step (evidence, index, seal,
mechanical attestation, transition) as completed while the plan does not
see it done is a `conflict` for `--resume` and for `--apply` without a
digest: progress is never a receipt, and the step is not replayed on the word
of an editable file. `--apply --digest <current plan>` — the plan the caller
reviewed — starts a new run and names the checkpoint it replaced.

**Start locks** (`internal/pose/start_lock*.go`). The lock is published with
its owner already in it (an exclusive temporary file hard-linked to the lock
path, which fails if the lock exists; exclusive create and write where links
are unavailable). The owner names the resource, worktree, operation, host,
application-specific digests of the machine and boot IDs (never the raw IDs),
PID namespace, PID and process start time. A lock held by another call is
`owner-active`, `owner-unknown` or `owner-dead`, never judged by age. Dead is
proven only on the same host and machine: another boot of it, or — in the
same boot and PID namespace — a PID that is not running or that runs a
process started at another time. Another host or machine, no machine ID,
another namespace, a lock with no readable owner (older engines wrote it
empty) or any platform but Linux is `owner-unknown`. `start --status` (CLI
and MCP) shows the state and the SHA-256 of the lock bytes.
`pose start spec:<slug> --recover-lock <owner digest>` removes a lock only
when, under an exclusive OS lock of the starts directory, its bytes still
have that digest and its owner is still proven dead; it never touches the
start record or the spec, so the interrupted intention resumes from its
record. A release removes the lock only if its bytes are still its own.
Lock files are runtime state: git-ignored in this repository and outside
the review subject.

## Consequences

- Journals, checkpoints and locks that used to read as valid or absent now
  stop the call that reads them, with a location and a next step. All twenty
  action journals in this repository and in Harne8 read under the new rules.
- Recovery is a person's decision or an explicit command naming what was
  inspected; nothing is deleted by inference.
- `pose start` gains `--recover-lock`; its new refusals and the
  `checkpoint-*` and `action-journal-invalid` causes are additions to
  documented contracts, classified minor. `StartStatus.lock`,
  `CloseoutPlan.checkpoint` and `data.replaced_checkpoint` are optional
  fields.
- Limits: liveness is provable only on Linux, for owners on the same machine;
  advisory and hard-link mechanisms protect cooperating writers only, not an
  editor; journals and checkpoints are not synced beyond what the writer did
  before (the journal append already syncs); power loss is not qualified.
  Coordination between start, close, transfer and editors is B14; atomic
  index generations are B15; request supersession recovery is B10; a daemon
  or runtime consumes this contract in its own qualification (B35) and is not
  presumed to be the owner of a lock taken before a restart.

## Falsifier

This decision is wrong if a journal line that does not read is ever part of
a view, if a checkpoint that does not read lets a step run, if a step a
checkpoint records but the repository does not reflect is run again without
the caller naming the current plan, or if a lock whose owner is running or
cannot be checked is removed by the engine. The regressions in
`internal/pose/recovery_integrity_test.go`, `recovery_lock_linux_test.go`,
`internal/cli/recovery_integrity_test.go`, `recovery_lock_linux_test.go` and
`internal/mcpserver/recovery_integrity_test.go` assert each of these.
