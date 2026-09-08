---
spec: pose-attestation-evidence-must-be-in-the-bundle
category: fixed
breaking: true
refs:
---

A criterion recorded `passed` must now cite evidence the sealed bundle actually
contains, of a class the criterion asks for. Nothing checked this before: a
`passed` could reference evidence absent from the bundle, evidence of the wrong
class, or nothing at all, and the attestation verified.

`pose review auto-attest` no longer synthesises a reference. Where the scope is
expected to carry validation evidence it refuses, naming the criterion, the
missing class and the way out; where it is not — a documentation-only spec with
no delivery target — it records the criterion `not-applicable` with a rationale
naming what is absent.

A criterion demanding an evidence class no registered check may emit now blocks
the plan, naming the criterion and the profile. Dropping the class — the
treatment tools receive — would be worse than the demand: a criterion left with
no class accepts any sealed evidence, so one asking for `test` would be
satisfied by a build result. The profiles shipped with POSE are reconciled to
classes that can actually be produced, so the blocker is a backstop rather than
a wall.

**This fails attestations that already verify.** They are unsupported and were
reported as sound; the remedy is a superseding attestation, which the
append-only model already provides. Plan digests change where an unproducible
profile was reconciled, so a bundle sealed before this does not match one sealed
after. An instance whose own profiles demand an unproducible class is blocked at
plan time until it reconciles them; `pose doctor`'s `review.evidence-vocabulary`
names which.

A completed scope keeps an approval recorded before the new rules through
`evidence_vocabulary_reconciled_at` in the review policy — a dated, opt-in
exemption in the shape `component_aware_adopted_at` already uses. Without it
this change failed 106 of POSE's own closeouts, and 52 after the profiles were
reconciled: work that was genuinely reviewed, failing to a rule that did not
exist when the review happened. The waiver covers only the evidence-support
checks; a malformed attestation is still rejected.

