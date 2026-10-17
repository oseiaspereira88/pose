---
title: A source that lost something is incomplete for what depends on it
status: accepted
date: 2026-10-17
decision_type: architecture
---

# A source that lost something is incomplete for what depends on it

## Context

Spec `pose-source-coverage-integrity` (B06; front 1 D01 and D13, front 2 T2-05, front 5
F05-01 and F05-03). Four readers return what they read and drop what they could not,
and every consumer takes the shorter list for the whole:

- **Follow-ups.** `ParseSpecFollowups` skipped a spec it could not read. With one
  readable follow-up and one spec that was a dangling link, `pose followups --all
  --json` exited 0 with one item and said nothing about the other file. Two readers
  also disagreed about what is unresolved: the backlog counted a bullet with no
  disposition and an explicit `[open]`, the obligation projection and the candidates
  only the explicit one, and the project-state provider counted only the explicit one
  (P01).
- **Obligations.** `ProjectObligations` appended "dropped an invalid projected
  obligation" to a limitation list and continued; the producer's coverage stayed
  `current` and `Complete` stayed true (D13).
- **Debt markers.** The scan of source files read each file through a `bufio.Scanner`
  whose `Err()` nobody read, so a line longer than the scanner's token stopped the file
  and the markers after it were lost with exit 0 (P04). The shared file scan also dropped
  directories it could not walk, files it could not read, files over the size limit,
  binary files and generated files, all without a trace.
- **Review history.** `ListReviewAttempts`, `ListReviewBundles` and
  `ListReviewAttestations` skipped a record they could not read or parse. A rejection
  that was truncated disappeared from the sequence, and the verification took the
  earlier approval as the current one (T2-05).

"Nothing found" and "could not look" are different answers; a zero must say which.

## Options considered

1. **Fail the whole read on any loss.** Honest, but a corrupt file in an old spec would
   stop every closeout of every other spec: the opposite of R4.
2. **A free-text limitation beside the list.** What the code does for dropped
   obligations today; a consumer cannot act on a sentence and `Complete` stays true.
3. **A typed coverage that travels with the result, attributed to a scope, with the
   consumers deciding what depends on it.**

Option 3 is chosen.

## Decision

**SourceCoverage** (`internal/pose/source_coverage.go`) is the account of one read:

- `Losses`: things that could not be seen. Each names the producer, the artifact, a
  `kind` (`read-error`, `parse-error`, `invalid`, `scan-incomplete`, `unsupported-format`),
  a detail that carries the operating-system or parser message and never bytes of the
  artifact, and, when it can be shown, the **scope** it belongs to and how that was found
  (`path`: the spec slug the file name gives; `header`: a readable prefix names the scope or
  the bundle). A loss with no scope is unattributed. `At` records when the lost record was
  made if its name says so.
- `Exclusions`: parts left out on purpose and declared, grouped by pattern with a count
  and examples (over the size limit, binary content, generated header). An exclusion is not
  a loss: a read with exclusions is complete for the slice it declares, and says that.

`CompleteFor(scope)` is true when no loss is attributed to the scope and none is
unattributed. A loss attributed to another scope does not count (R4: a corrupt spec stops
nothing outside itself); an unattributed one may belong to anything, so it does. An empty
scope asks about the whole source and sees every loss.

**Follow-ups.** `ReadSpecFollowups` returns the items and the coverage; `ParseSpecFollowups`
is the compatibility wrapper. `SpecFollowup.Unresolved()` is the one definition of what
still owes a disposition (explicit `[open]` or no disposition at all). The backlog (`pose
followups`, hence MCP `pose_get_followups`, which runs it), the candidates, the project-state
provider and the obligation projection all use it; the projection gives a bullet with no
disposition its own reason code, `followup-undisposed`. The CLI and the tool carry
`coverage_complete` and `coverage`; the human output names each unread file; `--fail-overdue`
cannot certify "none overdue" over an incomplete read and fails.

**Obligations.** A new producer state, `incomplete`: the producer ran and returned what it
read, but something it should have read was lost. The follow-up producer sets it when a spec
that concerns the query was lost (a query about another spec is unaffected); a dropped invalid
obligation sets it on the obligation's producer, so `Complete` is false and the detail says
why; the findings producer sets it when a review record cannot be read.

**Debt markers.** The scan reads the bytes the file scan already holds, line by line, with no
line limit. The report carries `scanned_files`, `complete` and `coverage`; the Markdown report
states the scope of the scan. The shared file scan returns its losses and exclusions.

**Review history.** `ReadReviewAttempts`, `ReadReviewBundles` and `ReadReviewAttestations`
return the sequence and the coverage. An unreadable record is attributed by its readable
prefix (a header scope, a `bundle_id`); one that names another scope or bundle does not concern
this one. `ReviewCheck` does not take the latest readable attempt as the current one while a
record that concerns the scope and is not provably older (the attempt file name carries its
time) could not be read: that is a blocker; a provably older one is a warning. With no
readable attempt at all, an unreadable one is "no readable attempt", not "none". Verification
does the same for attestations of the bundle: an unreadable one that may be a later rejection
blocks the earlier approval. Unreadable sealed bundles are disclosed as a warning, because an
approval is bound to the exact bundle that matched the current inputs. `pose review record`
refuses to choose the attempt it supersedes from a history with a hole.

## Consequences

- No reader returns an empty list for something it failed to read without saying so; the
  consumers named above carry the coverage in JSON and text. Compatibility: the added fields
  and states are additive; a repository with no unreadable source behaves as before.
- **Stricter, classified minor.** A repository that has an unreadable review record newer
  than its approval is now blocked until the record is restored or removed; a bullet with no
  disposition is now a projected obligation (it was already counted by the backlog); `pose
  followups --fail-overdue` fails over an incomplete read.
- Limits: attribution by path or by a readable prefix is a best effort and says how it was
  found; a record with neither is unattributed and treated as possibly concerning the scope. A
  sealed bundle that cannot be read is disclosed, not blocking, because verification is bound to
  the bundle that matches the current inputs. The tolerant sniff of a truncated record reads
  names, not decisions: nothing is reconstructed from bytes that were not read.

## Falsifier

This decision is wrong if a reader can return an empty or shorter result for a source it
failed to read without a loss in its coverage, if a loss attributed to one scope changes the
answer about another, if an unreadable record that may be later than the selected attempt or
attestation lets the earlier approval stand, if a scan that stopped early reports a complete
scope, or if the backlog, the candidates, the project state and Attention count different sets of
unresolved follow-ups. `internal/pose/source_coverage_integrity_test.go` and the CLI and MCP
parity tests assert each of these.
