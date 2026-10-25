# B16 prerequisite — delegated review attempt ledger

Canonical spec `pose-delegated-review-attempt-ledger`, existing R1–R4 preserved. Baseline `2c0bb3e`, branch `impl/pose-b16-attempt-ledger`. This closes the ledger prerequisite; `pose-delegated-review-run-continuity` remains separate. No parent pin update, remote publication, release, delegated-review adoption or external vendor call.

## Implementation and reproduction

The old CLI returned exit0 from history-check --strict after a tracked delegated run was deleted: synthetic reproduction `/tmp/pose-b16-repro.json`. It checked only report-history JSONL. Surviving-file scans and cached quota counts also could not witness lost attempts.

The implementation reserves one immutable ordinal slot before launch through the B14 exclusive synced publisher, rechecks current history/quota/reason, keeps an immutable receipt of the final run, and publishes a witness before transcript/run. CAS losers never launch or retry. Missing reservations/receipts/witnesses or corrupt records are explicit losses. Content-addressed runs retain schema1 with optional ledger_version1/attempt_id/reason; untouched legacy identities remain valid. New readonly fields disclose scope-wide runs alongside attestations. A single unambiguous completed, unsuperseded draft decision supplies declared telemetry only. Insights aliases stats and text/JSON/HTML/MCP share the aggregates; full history classifies reruns before the window limits counting.

Read inventories, JSON bytes, transcripts and Git observations are bounded. Writers check the same per-record bounds before publishing. Shared reader budgets cover run records and witness rereads; receipt/attempt phases have aggregate budgets. Tracked edits/deletions, including staged/committed ones, fail history-check. An unborn repository without delegated data keeps the old report-history contract.

## Actual separate review and fixes

Real separate execution `/root/b16_review` inspected domain/CLI/MCP and ran independent overlays/probes. Findings resolved with permanent regressions:

1. Missing reservation+receipt could reset quota while a failed run survived: enforce run-to-reservation identity and mandatory witnesses on ledger-version1 runs. Reviewer confirmed the fix; the independent probe compiled after the fix, so no independent before-fix execution is claimed.
2. Editing witness BundleID could evade a scoped reader: bind witness to run before filtering.
3. A time window erased prior attempts when classifying reruns: classify from all history, count only window events.
4. Writer could publish an oversized record its reader refused: align prepublication bounds; transcripts use bounded streaming hashing.
5. The first run-record loop's budget did not cover witness rereads: share the remaining budget and read each witnessed record once. Budgets in attempts/receipts are aggregated too.

Added four delegated adversarial-corpus cases backed by permanent tests. HTML metrics and the explicit insights CLI alias close presentation gaps. Initial full suite caught runtime directories absent on fresh install (registered them as optional runtime references), two old identity tests assuming exactly one loss (kept zero admitted runs and the identity-cause assertion while accepting witness diagnostics), and old history-only repositories without HEAD (preserve their behavior when no delegated data exists). Existing dispatch fixtures now state a reason for legitimate reruns and count actual launches, not quota refusals.

Reviewer identity `agent:independent-codex-b16-review` identifies this actual separate execution. No human confirmation, cryptographic identity or cognitive-independence claim. Native attestation follows canonical evidence and sealing, not this report. The primary runs complete suite/race/scanners; independent targeted runs are attributed to the reviewer only where actually executed.

## Rules and limits

Backend Go: shared domain and propagated errors. Security: exclusive immutable publication, declared labels kept separate from authority, no automatic vendor switching or execution of real vendor CLIs. Documentation style/delivery evidence: EN/PT mirrors, registered integration, immutable attribution and current provenance. Consumed knowledge:pose-b07-implementation and knowledge:adr-sealed-review-bundles-review. Integration inventory remains64contracts/1observed/63gaps; debt scan1covered/0uncovered is incomplete because two existing specs are invalid. No upstream contribution staged without consent.

Trust boundary: cooperative writers plus filesystem/Git anchors, not authenticated append-only storage. Whole unanchored evidence/witness erasure cannot be proved by the erased storage. Incomplete reservations remain blocking; the next continuity spec owns resume without a second quota or journal. Producer authentication and adapter containment remain their separate specs. Source data, baseline/start and prior requirements are not rewritten.

Full `TMPDIR=/var/tmp go test ./...`: PASS after the runtime-reference and history-only compatibility fixes. Expanded domain/CLI/MCP controls passed, including HTML and legacy identity. Targeted race passed domain/CLI (1.177s/8.811s) on the reservation implementation; final-budget race and canonical validation follow before sealing. Govulncheck found no vulnerabilities. The independent reviewer confirmed the final shared-budget implementation and adversarial probes without open findings.

## Native delivery evidence

Source commit `e20cf394cf2280b13140b939e8f53f08224086a4` carries the spec trailer. Final-budget race passed domain/CLI (1.433s/9.639s); pinned gitleaks found no secrets. Canonical validation passed 153/153 checks in 266.9s. Strict structural check passed with 30 preexisting warnings. Artifact reconciliation matched 35 claims/35 observed without errors; historical orphan warnings remain disclosed. After explicit index regeneration, surface-check reported 1 target/153 results/0 findings, provenance `sha256:aa2aafa2023a3e3c21fea89a8e23b50d8377ae687de2b425f8736ba50d51764a`.

Bundle `rvb-da193121009a577e` sealed 150 evidence records. The earlier reviewer reached its usage limit before attestation; no attestation is attributed to that execution. A new actual separate execution `/root/b16_final_review` independently inspected the implementation, reran domain/CLI/MCP controls (0.070s/0.339s/0.006s, all PASS), and recorded approved attestation `rva-83cfa3925739793c` as `agent:independent-codex-b16-final-review`, with 15 criteria and mapping `SD-d41323c1 → D1`. Verify reported ready-to-close, fresh and approved; review-check passed. Attribution is declared execution identity, without human confirmation or cryptographic identity assurance.

Native `pose close` applied status done and completed_at 2026-10-25. Required post-close discover/update-state completed. No follow-up disposition, capability adoption, parent gitlink, remote branch or release was changed.
