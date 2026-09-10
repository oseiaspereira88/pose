---
spec: pose-policy-keys-and-release-surface-coverage
category: added
breaking: false
refs:
---

Every policy the engine models now reports the keys it carries and the engine
does not read. It was the review policy alone, because that is where
`DisallowUnknownFields` had to be dropped; the exposure is identical in the
delivery, artifact, capability, docs, release and state policies, and identical
is the point — an operator who learns the finding exists for one file has no way
to know it does not exist for the next.

The keys come from the structs that read the file, so adding a field is enough
and the list cannot drift. A file read by two structs is held to the union:
`capabilities.json` is decoded once for the staleness thresholds and again for
the trigger thresholds, and either half alone would report the other's keys as
unread — the finding inverted, telling an operator to delete settings that work.
The `_comment` the shipped policies carry is not reported. The release policy is
examined wherever the engine reads it, including the location it still falls
back to.

Which policies are covered is checked rather than listed: a shipped policy is
either held to its keys or recorded, in writing, as having no type to derive
them from. `changelog.json` and `dor.json` are the two, and both would report
findings the engine itself causes.

The coverage audit that found three unreached branches in the doctor was run
over the rest of the CLI. The largest thing it found was that the entire release
command surface executed zero statements under the suite — including the rule
that release evidence must name the commit the tag points at, which is what
keeps a release record honest. That surface now runs under test, from 0 of 20
functions to 20 of 20.
