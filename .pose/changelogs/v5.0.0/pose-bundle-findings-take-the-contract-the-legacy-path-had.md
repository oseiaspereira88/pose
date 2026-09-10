---
spec: pose-bundle-findings-take-the-contract-the-legacy-path-had
category: fixed
breaking: true
refs:
---

Validating an attestation against a sealed bundle now applies the finding
contract the legacy attempt path always applied.

It blocked a finding only when its disposition was `open` or
`changes-requested`. Everything else passed with no blocker: an accepted risk of
severity `critical` with no owner, no rationale and no review date; a disposition
the engine does not know; a finding with neither severity nor action. And two
policy settings were not consulted on this path at all —
`allow_approved_with_reservations` lost to a literal check for `approved`, so a
project that enabled reservations found them refused anyway, and
`accepted_risk_severities` decided nothing.

A project that adopted review bundles therefore lost a gate it had before, and
two settings quietly stopped taking effect. Review bundles exist to make review
more rigorous; on this axis they had made it less.

The two policy-dependent parts are sealed into the bundle rather than read live,
for the reason the governance contracts already are: a flag flipped today would
otherwise approve a closeout recorded years ago. The rest is not configuration —
a finding without a severity is incomplete under any policy — and applies to
every bundle, including those sealed before the field existed. Nothing already
recorded is re-judged: the 469 attestations in this repository carry no findings
at all.

The sealed payload changes, so a bundle sealed by this release does not digest
the same as one sealed before it.
