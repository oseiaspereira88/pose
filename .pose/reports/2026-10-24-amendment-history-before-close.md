# Amendment history before close — implementation and review

Date: 2026-10-24. Canonical spec: `pose-amendment-history-holds-before-close` (existing R1–R3, not a new copy). Baseline `b79744d2f3eda1cbb93435813d65b026f1fa845a`, branch `impl/pose-amendment-history-before-close`. B11 requires this prerequisite first; B11 itself remains separate. No remote publication, release, adoption or parent gitlink update.

## Reproduction and correction

Before the fix, TestAmendmentHistoryFirstChangeSeedsRecordedStart failed for both schema variants (only the change event existed), TestAmendmentHistoryInProgressWarnsBeforeDone had no warning, and TestAmendmentHistoryCloseRefusesBeforeTransition observed exit 0 / Closed despite invalid history. Local reproduction log `/tmp/pose-amendment-repro.log` is supplementary, not canonical delivery evidence.

Root cause: amendment writing had no bridge from the atomic start snapshot, and history comparison lived only in done lint. The fix copies validated recorded start nodes into a new baseline event before the first valid non-baseline change. Schema 1 filters requirements, schema 2 preserves node state/relations. Current timestamp/declared author are retained, with exact start slug/plan digest/RecordedAt in rationale; no past review or signing is invented. A shared readonly history assessment serves lint and pre-transition close; close repeats it under the lifecycle lock shared with amend. Existing journals are never rewritten/repaired. No journal retains the opt-in contract.

## Validation and review

Expanded domain/CLI regressions passed, including new/changed/transition nodes, flat specs, malformed/incomplete/wrong-spec/tampered start, no-start/existing history, zero journal writes for refused requests, positive close and journal mutation after initial gates. An invalid verified-assumption fixture initially failed for missing Evidence/Scope; the fixture was corrected to comply with the existing contract, and the rerun passed. Catalog/MCP tools are unchanged; existing readers consume the same journal schemas.

Actual separate review execution `/root/amendment_review` inspected R1–R3 and ran amendments/contract-node/trace/close-revalidation regressions: PASS. No blocking findings. Its low operational suggestion to identify the start precisely in seed rationale was applied. Reviewer did not execute the primary's complete suite, race or scanners and does not claim so. Native declared identity `agent:independent-codex-amendment-review` identifies that actual agent execution, with no human confirmation, cryptographic identity or cognitive-independence claim. Native attestation follows canonical evidence and bundle sealing.

## Boundaries

Explicit `--baseline` deliberately keeps its current-state snapshot contract. Hash checks do not prove materiality, rationale or review truthfulness. Old/incomplete journals require explicit repair; append/Sync failure is reported without rollback or truncation. Shared writers need the same lifecycle locking behavior. Generic adversarial corpus expansion is a low-priority open follow-up, not a claim that the permanent new regressions are absent. The accepted ADR records this compatibility change.

## Verification notes

`govulncheck ./...` passed without vulnerabilities. The initial full suite caught the direct-print ratchet improvement (amend.go prints dropped 21 to 19); the tracked baseline was lowered as required. The same concurrent collection also encountered existing ambient Git fixture status-128 failures in unrelated review tests. No Git engine code was changed; final canonical validation uses isolated TMPDIR/GIT_CEILING_DIRECTORIES and sequential producers. Full/race outcomes are recorded only after their successful reruns.

## Rules applied during review

Backend Go: domain gate reuse, propagated errors and cooperative lock. Security: no secrets or automatic policy/authority changes. Documentation style: canonical spec and EN/PT manual parity. Delivery evidence: registered integration required before close. Knowledge: consumed sealed-bundle decision log and B07 handoff; no upstream contribution staged without consent. Tech-debt assessment remains incomplete due to two preexisting invalid spec sources; one covered marker and zero uncovered do not prove global absence of debt.
