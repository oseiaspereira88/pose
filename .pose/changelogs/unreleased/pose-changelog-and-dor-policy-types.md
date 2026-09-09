---
spec: pose-changelog-and-dor-policy-types
category: added
breaking: false
refs:
---

`changelog.json`'s `categories` is now the setting it always looked like. It
shipped from the start and nothing read it: the set of valid fragment categories
was written out three times instead — in the fragment loader, in `pose check`,
and implicitly in the notes renderer. Configuring it now governs all three, and
the refusal an author reads quotes the configured set rather than a list written
beside it.

An absent policy, or one declaring no categories, keeps the six defaults, so an
instance that never touches the file behaves exactly as before. A category a
project adds renders under a heading of its own, after the known ones. The
section order itself stays in code: putting `security` first is a reading
decision, not a configuration one.

`dor.json` now says that the Definition of Ready is opt-in. `readiness.go`
looked for `adopted_at` in a file that never shipped it, so the gate read as
unadopted everywhere — which is the intent, but an omitted key is
indistinguishable from an oversight. The policy carries an explicit empty date
and a comment saying what setting it does. No instance changes behaviour: empty
is what absent already meant.

Both files are read through one named type each, covering the whole file rather
than the half a given command needed, and both are now held to their keys by
`pose doctor` — the exemption list that named them is empty.
