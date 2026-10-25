# B16: Delegated run continuity

## Scope and basis

Canonical spec `pose-delegated-review-run-continuity`, prerequisite ledger done at `3fa69b6`. Preserve R1–R9 and reuse its immutable reservations, witnesses and exclusive publisher. Consumed knowledge:pose-b07-implementation and knowledge:adr-sealed-review-bundles-review. No vendor command or upstream publication was invoked for this feature; test adapters are synthetic shell fixtures.

## Implementation

Add declared RE2 limit/session/reset recognition on both supported output channels, private session material, same-attempt continuation links and exclusive predecessor claims with random nonces. Revalidate original predecessor, bundle, brief, policy, reconstructed disk snapshot and budgets before launch. Redact captured session references before publishing transcript/output; ambiguous or rotating captures withhold bytes. Session placeholders must be separate argv values and cannot occupy executable/interpreter-code slots. Failed launch is failed, observed process loss or timeout is interrupted (unknown without declarations), and interrupted worktree mutation fails rather than becoming a review.

A duplicate resume returns its existing result or an identified pending continuation without starting another process. Claims/results/runs stay immutable. Reader bounds and history checks include continuation artifacts. Late supersession preserves every record but counts one execution endpoint. Resume counters remain distinct from the original reservation/quota. Only completed, unsuperseded draft decisions affect declared rates; interrupted drafts are partial and never rejecting reviews. CLI status/verify/doctor and MCP summaries disclose safe metadata. Explicit cross-adapter fallback from a pending usage-limit endpoint records limit-of and both run identifiers; the engine never chooses an adapter.

## Actual separate review

Actual separate execution `/root/b16_continuity_review` inspected implementation, rules and ADR, executed independent targeted tests and overlays, and identified fixes for identical-time claim collision, failed-writer interruption metadata, capture/redaction on output-only drafts, incomplete continuation receipts, session rotation, launch failure classification, supersession accounting, executable session interpolation and historical fallback selection. Findings by inspection are not claimed as before-fix execution. Output-only leakage was independently reproduced before its fix; final output/receipt/writer overlays passed. Independent targeted controls passed domain 0.012s/CLI 1.363s; the earlier MCP targeted invocation had no tests to run, so it is not claimed as MCP continuity evidence. A permanent MCP partial-draft control was added subsequently. No attestation is claimed before canonical validation and bundle sealing.

## Validation and limits

Expanded dispatch/ledger/snapshot/continuity tests passed. Full Go suite passed after registering optional reviewers policy references for fresh installations. Initial race domain/CLI passed; final race including actual concurrent resume launches, observed process loss and MCP follows before sealing. Final canonical validation, artifact reconciliation and native gates remain pending.

Backend Go, security, documentation-style and delivery evidence rules applied. Independent debt assessment is incomplete due to two preexisting invalid specs (one covered marker, zero uncovered); integration inventory remains 64 contracts/1 observed/63 gaps. No global debt or integration completeness is claimed.

Sessions are private local runtime, not portable authenticated identity. Losing private material prevents recovery. A lost host before publishing a run/result remains explicit incomplete evidence; a pending claim is returned without blindly relaunching. No recovery or completed review is fabricated without a captured interrupted record. Git/filesystem anchors remain trusted, with no cryptographic append-only claim. Default limits are three resumes and 60 minutes accumulated wall-clock; wait is one bounded reset plus at most one second jitter, never vendor polling. No capability adoption, upstream contribution, parent gitlink or remote merge is part of this delivery.
