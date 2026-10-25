# ADR: Reserve delegated attempts before launch and witness immutable runs

## Status
Accepted (2026-10-25), spec `pose-delegated-review-attempt-ledger`.

## Context
Existing runs are content-addressed, but deletion leaves no trace in a scan of surviving files, and a preview's cached run count races another dispatch. B14 provides the exclusive synced publisher. The accepted delegated adapter ADR separates drafts from verified attestations; B17/B19 own signatures and governed conclusions, and run continuity is a following spec.

## Decision
Before launching a valid dispatch, reserve an immutable ordinal slot per bundle under `.pose/review-attempts/`, using authoritative PublishFile ExpectAbsent. The content-addressed attempt ID includes a nonce; one contender wins a slot, losers refuse without launching or retrying. Re-read current runs and reservations at execution, enforce per-adapter quota including reservations, and require a bounded explicit reason whenever any completed run exists for the unchanged bundle, across adapters/kinds. An incomplete reservation blocks another attempt and is never auto-deleted or declared completed. An immutable result receipt links its final run after dispatch finishes. No mutable in-flight state or lock-order dependency is introduced; crashes remain explicit incomplete records for the continuity spec.

Before publishing a run/transcript, publish its immutable content-hash witness under `.pose/review-run-ledger/`. Use the same exclusive synced publisher for all three files; retries of identical bytes are idempotent, conflicting bytes refuse. A witness whose run or transcript is missing/corrupt is a visible incomplete history, not discarded. Validate record bounds and identities before reading; bounded inventories refuse rather than truncate. Legacy schema-1 runs stay readable without new fields or invented receipts. New optional fields do not change their historic identity.

History-check compares live witnesses and receipts and checks Git for modification/deletion of already-tracked immutable run/ledger/attempt artifacts, including staged and committed rewrites. Whole untracked history erased before anchoring cannot be proved absent; disclose this filesystem/Git trust boundary, do not claim authenticated append-only storage. Adversarial control of all witnesses and Git is outside that boundary.

Verify discloses scope-wide earlier runs beside the current attestation. Insights exposes per-bundle counts, per-adapter declared draft approval rate and same-bundle reruns, with coverage loss explicit. Only a single unambiguous `decision: approved|changes-requested|rejected` line classifies a completed draft; unknown/ambiguous output stays unknown and failed/stale runs are not rejections. These labels never authenticate vendor/model or apply an attestation. No new policy/capability adoption, external vendor invocation or automatic fallback.

## Consequences
Breaking stricter rerun admission; add --reason without changing existing positional API callers (optional trailing reason). Failed attempts consume quota; incomplete reservations require explicit inspection. Existing legacy records remain declared history. No journal cleanup, silent repair, proof fabrication or infinite retry. Rejected alternatives: surviving-file scans lose deletions; holding a lock through a long reviewer process leaves brittle ownership; a mutable counter loses crash evidence; automatic migration would invent past reservations. Future continuity reuses attempt IDs/receipts rather than introducing another quota.
