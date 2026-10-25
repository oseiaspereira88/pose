# B16: Delegated run continuity

## Scope and basis

Canonical spec `pose-delegated-review-run-continuity`, prerequisite ledger done at `3fa69b6`. Preserve R1–R9 and reuse its immutable reservations, witnesses and exclusive publisher. Consumed knowledge:pose-b07-implementation and knowledge:adr-sealed-review-bundles-review. No vendor command or upstream publication was invoked for this feature; test adapters are synthetic shell fixtures.

## Implementation

Add declared RE2 limit/session/reset recognition on both supported output channels, private session material, same-attempt continuation links and exclusive predecessor claims with random nonces. Revalidate original predecessor, bundle, brief, policy, reconstructed disk snapshot and budgets before launch. Redact captured session references before publishing transcript/output; ambiguous or rotating captures withhold bytes. Session placeholders must be separate argv values and cannot occupy executable/interpreter-code slots. Failed launch is failed, observed process loss or timeout is interrupted (unknown without declarations), and interrupted worktree mutation fails rather than becoming a review.

A duplicate resume returns its existing result or an identified pending continuation without starting another process. Claims/results/runs stay immutable. Reader bounds and history checks include continuation artifacts. Late supersession preserves every record but counts one execution endpoint. Resume counters remain distinct from the original reservation/quota. Only completed, unsuperseded draft decisions affect declared rates; interrupted drafts are partial and never rejecting reviews. CLI status/verify/doctor and MCP summaries disclose safe metadata. Explicit cross-adapter fallback from a pending usage-limit endpoint records limit-of and both run identifiers; the engine never chooses an adapter.

## Actual separate review

Actual separate execution `/root/b16_continuity_review` inspected implementation, rules and ADR, executed independent targeted tests and overlays, and identified fixes for identical-time claim collision, failed-writer interruption metadata, capture/redaction on output-only drafts, incomplete continuation receipts, session rotation, launch failure classification, supersession accounting, executable session interpolation and historical fallback selection. Findings by inspection are not claimed as before-fix execution. Output-only leakage was independently reproduced before its fix; final output/receipt/writer overlays passed. Independent targeted controls passed domain 0.012s/CLI 1.363s; the earlier MCP targeted invocation had no tests to run, so it is not claimed as MCP continuity evidence. A permanent MCP partial-draft control was added subsequently. No attestation is claimed before canonical validation and bundle sealing.

## Validation and limits

Expanded dispatch/ledger/snapshot/continuity tests and the full Go suite passed after registering optional reviewers policy references for fresh installations. Final race including concurrent resume launches, observed process loss and MCP passed. Canonical validation, artifact reconciliation and native closeout passed as recorded below.

Backend Go, security, documentation-style and delivery evidence rules applied. Independent debt assessment is incomplete due to two preexisting invalid specs (one covered marker, zero uncovered); integration inventory remains 64 contracts/1 observed/63 gaps. No global debt or integration completeness is claimed.

Sessions are private local runtime, not portable authenticated identity. Losing private material prevents recovery. A lost host before publishing a run/result remains explicit incomplete evidence; a pending claim is returned without blindly relaunching. No recovery or completed review is fabricated without a captured interrupted record. Git/filesystem anchors remain trusted, with no cryptographic append-only claim. Default limits are three resumes and 60 minutes accumulated wall-clock; wait is one bounded reset plus at most one second jitter, never vendor polling. No capability adoption, upstream contribution, parent gitlink or remote merge is part of this delivery.

## Native delivery evidence

Source commit `982f2c348cbb8836210ce2e8fbb1346ef9fa15e2` carries the spec trailer. Full Go suite passed; final race controls passed domain 1.023s/CLI 3.373s/MCP 1.026s, including actual eight-way resume contention with one launch and observed SIGKILL interruption. Govulncheck found no vulnerabilities; pinned CI gitleaks found no secrets in the source commit. The first scanner invocation used the wrong config-relative directory and did not scan; the corrected invocation used the CI directory/config and passed.

Canonical validation passed 154/154 checks in 245.9s. Strict structural check passed with 30 preexisting warnings. Artifact reconciliation matched 32 claims/32 observed without errors; historical orphan warnings remain disclosed. After explicit index regeneration, surface-check reported 1 target/154 results/0 findings, provenance `sha256:2b90542a3f5a622c2773df0cb1b32e069eaf32d6243a307de3298a57c6401a90`.

Actual separate execution `/root/b16_continuity_review` reconfirmed the committed source and ran final independent domain/CLI/MCP controls (0.009s/1.670s/0.006s, all passed), including the permanent MCP partial-draft case. Bundle `rvb-a2a8de43595b6c2d` sealed 151 evidence records. The reviewer recorded approved attestation `rva-2c1e6fda7692c955` as `agent:independent-codex-b16-continuity-review`, with 15 criteria, 11 tool dispositions and mapping `SD-471b95ed → D1`. Its initial apply preview refused without writing because required tool dispositions were missing; the completed attestation supplied them. Verify reported ready-to-close, fresh and approved; review-check passed. Identity and separation are declared, without human confirmation or cryptographic assurance.

Native `pose close` applied done/completed_at 2026-10-25. Required post-close discover/update-state completed. No follow-up disposition or remote publication was changed.

## Requested remote-main verification

On 2026-10-25, fetched origin/main of `oseiaspereira88/pose`. It was `27c5be7a8dcb3226ae8d49e45af48bbc289a9f5c`, merge PR #204 (B05). Ledger source commit `e20cf394cf2280b13140b939e8f53f08224086a4` was not its ancestor; the ledger spec there remained draft and its implementation files were absent. Therefore the ledger implementation had not been integrated into remote main. The fetched Harne8 remote main still pinned pose-dist at `36fa2b6dab817e202bce3bee82b083af3640c56d`. No push, PR or merge was performed for this verification.
