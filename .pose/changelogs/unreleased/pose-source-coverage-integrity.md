---
spec: pose-source-coverage-integrity
category: changed
breaking: false
---

A read that could not see part of its source now says so instead of reporting less.
`pose followups` (and the MCP tool `pose_get_followups`, which runs it) lists the specs it
could not read in `coverage` and prints them in the human output; a spec that cannot be
opened no longer makes the backlog look shorter, and `--fail-overdue` fails instead of
certifying "none overdue" over an incomplete read. The backlog, the candidates, the project
state and Attention now count the same unresolved follow-ups: a bullet with no disposition
is unresolved everywhere, and Attention projects it with the reason `followup-undisposed`.

The obligation report gains the producer state `incomplete`: an obligation refused by
validation, a spec or review record that could not be read, now make the report incomplete
for the producer they belong to, not just add a line of text. A query about one spec is not
made incomplete by a loss in another.

`pose assess tech-debt` finds a marker after a line longer than 64 KiB (the scan stopped at
such a line and reported none) and reports what it scanned: files it could not read make the
report incomplete, and the files it leaves out on purpose (over the size limit, binary,
generated) are counted by reason.

A review record that cannot be read is no longer skipped. `pose review-check` and `pose review
verify` do not take the latest readable attempt or attestation as the current one while an
unreadable record that may be later (or whose bundle or scope it names) exists; one whose
name shows it is older only warns, and unreadable sealed bundles are disclosed. `pose review
record` refuses to choose the attempt it supersedes from a history with a hole.
