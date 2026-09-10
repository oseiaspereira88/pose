---
spec: pose-reuse-is-sealed-signing-stays-live
category: changed
breaking: false
refs:
---

`allow_criterion_reuse` is sealed into the review bundle.
`require_signed_attestations` stays read from the live policy, deliberately, and
is now the only gate a sealed bundle is judged by that is not sealed with it.

The two look alike and point opposite ways. Reuse carries a prior criterion
disposition into this attestation, so whether it was permitted is a property of
the review that happened — read live, a project enabling reuse today
retroactively legitimises an attestation that reused a criterion when its own
policy forbade it.

The signing requirement is a bar rather than a permission. A project that starts
requiring signed attestations is raising it, and a bundle sealed before that must
not be permanently exempt; sealing it would mean the new requirement applies only
to future work, which is the opposite of adopting it. The exception is annotated
where the code makes it, not only in the ADR.

A bundle sealed before this gate refuses reuse, which is the conservative reading
and matches what an instance with reuse disabled already got.
