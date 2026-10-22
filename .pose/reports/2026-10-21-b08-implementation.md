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
  the initial implementation. Final corrected-source execution passed on 2026-10-22 (pose 64.960s, CLI 73.336s, MCP 5.486s).
- Preliminary native validations each passed 139/139, first at 7d73ae8 and then
  c8f95e0. They precede final review corrections and are not final delivery evidence.
- Final module collection passed 139/139 at a14e5ad on 2026-10-22. Global collection
  then passed 142/142 at that same committed source, including mcp-enforce, with
  no module filter. Canonical evidence is `.pose/results/delivery-validation.json`.
  Collector log `/tmp/pose-b08-native-validation-global.log` reports 163.6s.
- Strict structure check passed with 29 existing warnings. Artifact reconciliation
  found zero error findings. Surface check passed: one capability, 142 results, zero findings, scoped provenance
  sha256:091f0863ffbe6b2b4310143569ee5ce2e08c6f21d215caab2b7eb6b14af69aec.
  Sealed bundle rvb-50747fccf11f344f contains 139 observed evidence entries relevant
  to pose-mcp (global validation additionally covered mcp-enforce). Its implementation
  digest is sha256:cf91f10c7af3102fedbba0b0994d88d7456914e6439983096198640a9def2614.
  Final R3 independently approved the sealed implementation subject (actual session
  01a161dc-78bf-77a1-bcc2-2040337755a2), without reclassifying R2's old reservation.
  Attestation rva-259365f3e469f543 records the real R3 conclusions, with explicit
  prepared/concluded/applied roles and no fabricated confirmation. Native review
  verification returned fresh/approved/ready-to-close, review-check passed, and
  `pose close spec:pose-action-target-and-node-scoping` applied done on 2026-10-22.
  Post-close discovery reused all three components against unchanged content,
  engine and matrix. Post-close native check passed in strict mode with29 existing warnings;
  lint-spec and history-check passed, closeout-check reports lifecycle_done=true,
  review_approved=true, terminal=true, next_action=none. Index generation
  gen-8c5963c8463bb53e verifies current before the metadata commit.
  Native state history/refresh-log are derived operational closeout outputs outside
  the artifact policy's governed roots; they are retained with their actual events.

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
actual MCP unavailability/authorization loss. R2 is a fresh separate execution on that changed subject, session
01a160ad-bf20-7252-8478-3e90e85ac105. It confirmed both fixes and found no further
source defect, with reservations about final gates and semantic adequacy of the
selected subject; final adjudication will be appended to its actual report. No human, cross-vendor or cognitive-independence
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


## Final adjudication

[Actual R3, appended after the untouched R2 report](2026-10-21-b08-review-r2.md)
approves the exact sealed subject. It inspected collected evidence and reproduced
bundle identity, but did not execute tests. The sealed139 entries are 138 observed
passing pose-mcp checks plus the structural assessment; the global142 collection
contains139 pose-mcp checks (vet is not selected as sealed evidence) and three
mcp-enforce checks. R3 observed an unstaged history update during review; the lead
subsequently staged it and history-check passed (108 staged/clean, zero untracked
or unstaged). Stage/verify again after applying closeout outputs. Artifact scan has
zero errors with652 historical orphan/provenance warnings, not a warning-free repo.

The final reviewer is declared under an alias tied to its real session; the
implementation agent prepares/applies the record, the separate execution concludes,
and no human confirmation or cryptographic actor identity is fabricated.
