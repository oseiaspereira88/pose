# ADR: Continue delegated runs under one immutable attempt

## Status
Accepted (2026-10-25), spec `pose-delegated-review-run-continuity`.

## Context
The attempt ledger coordinates dispatch, but an interrupted vendor session needs a continuation rather than a new quota reservation. Run evidence and predecessor receipts cannot be rewritten. Vendor classification belongs in policy, and session references must not enter transcripts or reports.

## Decision
Add optional adapter interruption/session/resume declarations. Recognize usage limits only through declared exit codes or RE2 patterns; capture reset instants as RFC3339 and session references through an explicit capture group. Store session material privately outside the worktree and disclose only its digest. Redact captured references before persisting output/transcript. Undeclared interruptions remain unknown and not resumable.

Link each continuation to its immutable predecessor under the same attempt. Reserve an exclusive per-predecessor continuation claim before launch, with a result receipt after publication. A duplicate returns the existing result, or reports its pending claim without launching again. Never clear a claim after a crash or invent a completed run. Reconstruct the original brief from recorded notes and compare its digest, pinned adapter policy and sealed checkout snapshot. Enforce a default three resumes and accumulated process wall-clock budget. Reuse the existing supervisor, publisher and containment checks.

## Consequences
Interrupted drafts remain partial and cannot authorize review effects. Attempts and resumes have separate counters. Wait only for a declared bounded reset, with bounded jitter and no vendor polling; revalidate after the wait. Explicit cross-adapter dispatch carries `limit-of:<run>` and discloses both runs. Filesystem/Git anchors remain the ledger trust boundary. A claim without a result remains an explicit incomplete operation; restarting the same predecessor never launches another process. Rejected alternatives: a reason-bearing new attempt double-counts recovery; mutable predecessor updates erase evidence; automatic vendor fallback changes declared independence.
