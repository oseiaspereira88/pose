---
spec: pose-journal-checkpoint-and-lock-recovery
category: changed
breaking: false
---

Action-request journals are read whole. A line with a second JSON value, a
duplicate field, an unsupported `schema_version`, a sequence gap, an empty line
or no line terminator used to be read as if it were not there, or made the
request unreadable without saying where; it is now an `action-journal-invalid`
error naming the line, byte offset, length and SHA-256 of the bytes, and
whether it is an interrupted tail or corruption. `pose action show|list` show it
in both renderers, MCP returns `error_code: action_journal_invalid` with the
same location, and an interrupted tail comes with a truncation proposal for a
person to review. The engine never truncates a journal and refuses to append
to one that does not read. Journals already written read as before.

A closeout checkpoint that does not read, has another schema or names another
scope is no longer read as absent: `pose close --plan` shows its state
(`checkpoint` in JSON), and `--apply`/`--resume` stop before any step with
`checkpoint-corrupt`, `checkpoint-unsupported` or `checkpoint-unavailable`. A
checkpoint that records an evidence, index, seal, attestation or transition
step the repository does not reflect is a `checkpoint-progress-mismatch`
conflict instead of being resumed; `--apply --digest <current plan>` starts a
new run and reports the checkpoint it replaced. The checkpoint is replaced
whole, and a removal that fails at the end of a run is reported.

The start lock records its owner. A lock another call holds is reported as
`owner-active` (`start-in-progress`), `owner-unknown`
(`start-lock-owner-unknown`) or `owner-dead` (`start-lock-orphaned`), never
broken by age; `pose start spec:<slug> --status` (and MCP `pose_start_status`)
show it, and the new `pose start spec:<slug> --recover-lock <owner digest>`
removes a lock whose owner is proven dead on the same Linux machine, leaving
the start record to resume. Locks left empty by earlier engines read as
`owner-unknown` and are removed by a person, as before.
