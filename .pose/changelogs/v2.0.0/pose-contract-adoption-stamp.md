---
spec: pose-contract-adoption-stamp
category: fixed
breaking: false
refs:
---

`pose update` now records `evidence_vocabulary_reconciled_at` in an instance's
review policy when the key is absent, dated to the update. Without it an
instance received the stricter attestation rules and no statement of when they
arrived, so every closeout recorded under the previous contract failed —
106 of them in POSE's own repository before the date was added by hand.
`.pose/policy/` is not machinery, so nothing else was going to deliver it.

An existing date is never touched, including one deliberately cleared by a
project that wants its whole history judged by the current contract, and a
policy the engine cannot parse is not rewritten. The stamp is reported in the
update's output.

`pose doctor` gains `review.contract-adoption`, which reports an instance
carrying recorded reviews and no reconciliation date, rather than leaving the
operator to read a wall of failing closeouts.

A `pose update` that replaces the binary now hands off to it instead of
finishing in the old process, so a migration shipped in a release applies on
the update that delivers it rather than the one after. And the adoption date is
inclusive of its own day: a review recorded at 09:00 was recorded before the
15:00 update that brought the contract.

