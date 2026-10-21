# B08 implementation and validation — 2026-10-21

Scope: spec:pose-action-target-and-node-scoping. Base 4fb8601 (merged PR195).
Worktree `/tmp/pose-b08-work`, branch `impl/pose-b08-action-target-and-node-scoping`.
No Harne8 gitlink, policy adoption, release publication or unrelated branch change.

## Implementation

Use one domain selector for queries and governed effects. Separate provenance,
relevance, effects and material subject. Keep the documented empty-source default;
expand current membership and preserve known members alongside partial coverage.
Validate exact node tuples and recompute subjects in their recorded artifact.
Use transport-supplied foreign authorization; never infer a local global blocker
from unknown membership or include foreign-only destinations in all-local gates.
Add structured effects through CLI/MCP while preserving legacy strings and journal
digests. Newly created approvals/acceptances under adopted agency readiness require
subjects; legacy unbound journals expose their lack of content-change detection.
Changelog marks compatibility tightening as breaking, with release review deferred.

## Executed evidence

- Initial three regressions failed on the base: wrong-origin restriction, subject
  hash from the origin instead of its own artifact, and hidden terminal-origin request.
- Targeted TestActionTarget suites pass across domain, real CLI and HTTP MCP.
- Full Go suite initially exposed invalid material fixtures and missing distributed
  manual synchronization; fixtures and assertions were corrected without removing
  gate/proof tests, Portuguese documentation updated, and scaffold regenerated.
- Race detector over internal/pose, internal/cli and internal/mcpserver passed on
  the initial implementation. Final corrected-source execution is pending.
- Preliminary native validations each passed 139/139, first at 7d73ae8 and then
  c8f95e0. They precede final review corrections and are not final delivery evidence.
- Final native validation, surface/review verification and closeout remain pending.

Commands use `GOTOOLCHAIN=go1.26.10`,
`TMPDIR=/var/tmp/pose-b07-validation-0bf729f`, `GOFLAGS=-buildvcs=false` for validation.
The isolated temporary directory avoids externally created `/tmp/.git` contaminating
fixtures. Disabling VCS stamping applies to validation builds, not release workflows.

## Separate review

The Claude attempt hit its weekly limit before reviewing anything (session
15fe8484-974a-48de-b6b0-caafd6a73f51). Waiting for the weekly reset would interrupt
this authorized delivery. The secondary-agent skill permits a same-vendor fallback.
R1 was a separate Codex execution, task-designated gpt-6.1-sol, session
01a160a0-2a2f-73e2-ba21-477c62cb2b36. Its [actual report](2026-10-21-b08-review-r1.md)
requested two fixes: retain known members despite malformed entries; disclose
unknown foreign membership even if relevance can no longer be established.
Commit 71dae1b addresses both with regressions, plus other-artifact mutation and
actual MCP unavailability/authorization loss. R2 is a fresh separate execution on
that changed subject, pending. No human, cross-vendor or cognitive-independence
claim is made. Read-only reviewers cannot run Go fixtures requiring writes; supplied
validation is attributed to the lead's actual collector, not reviewer execution.

## Assessments and limits

Integration assessment ran and retains existing heuristic provider/consumer gaps.
Tech-debt assessment found one covered TODO, zero uncovered markers, but remains
INCOMPLETE because two legacy spec sources are invalid:
pose-instance-engine-version-tracking and pose-knowledge-durable-reference-type.
This is not evidence that the entire repository is debt-free. Native check initially
found a malformed changelog header, corrected before final validation.
Partial node scope reports the named restriction without proving independence.
Unknown membership limits observation; it does not authorize fabricated destinations.
Subject binding proves the declared material identity, not semantic adequacy of a
human decision. B09 authority/waiver, B10 recovery and B11 snapshots remain out of scope.
