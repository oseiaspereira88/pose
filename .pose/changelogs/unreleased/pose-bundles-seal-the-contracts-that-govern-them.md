---
spec: pose-bundles-seal-the-contracts-that-govern-them
category: changed
breaking: true
refs:
---

A sealed review bundle now names the governance contracts that were in force when
it was sealed, and verification asks the bundle instead of asking today's policy.

The exemption that keeps a finished scope's approval when a contract arrives
after it compared the review's timestamp against `contract_adoptions` in the live
policy. That date is editable and re-read on every verification, so moving it
forward retroactively exempted more historical closeouts — an immutable bundle
judged by today's configuration, which is the objection this repository already
accepted when it sealed `selected_profiles` rather than reading profile selection
live.

Sealing the set also removes the need for a new dated marker per contract, and
gives a property the date could not: a bundle sealed by an engine that did not
know a contract never lists it, permanently, and no later edit reaches it.

A bundle sealed by this release does not digest the same as one sealed before it.
Bundles already sealed carry no list and keep the dated reading — they cannot be
back-filled, because rewriting an append-only immutable artifact is what sealing
exists to prevent and re-sealing would break the attestations bound to their
digests. The dated rule is therefore demoted rather than removed: it governs only
bundles predating the field, and the legacy attempt path, which has no bundle at
all.

This covers the contracts. Verification still reads six other policy fields live,
and each needs its own answer — tightening a signing requirement probably should
reach an old bundle. That is recorded as a follow-up rather than settled.
