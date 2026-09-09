---
spec: pose-doctor-reports-unread-policy-keys
category: added
breaking: false
refs:
---

`pose doctor` reports any top-level key in `.pose/policy/review.json` that the
engine does not model, and lists the keys it does read.

The decoder stopped refusing unknown keys in 2.0.2, so a newer field never
breaks an older binary reading the same repository. The cost was that a
misspelling — `contract_adoption` for `contract_adoptions`,
`allow_criteria_reuse` for `allow_criterion_reuse` — is silently ignored, and
the file reads as configured while the setting does nothing. This is a finding,
not a refusal; the refusal is what was removed on purpose.

The known-key list is derived from the policy struct, so it cannot drift into
reporting a real key as unknown.
