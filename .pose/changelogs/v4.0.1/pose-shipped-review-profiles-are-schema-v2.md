---
spec: pose-shipped-review-profiles-are-schema-v2
category: fixed
breaking: false
refs:
---

`milestone-integration.json` and `roadmap-outcome.json` ship at schema v2. They
shipped at v1, and the v1-to-v2 migration copies the distribution's own profile
over the instance's — so it copied a v1 file over a v1 file and logged
`review-profile (migrated): <name> (v1 -> v2)` on every `pose update`, forever,
telling the operator a migration had succeeded that never happened.

Two costs, not one. The instance never reached v2 for those profiles, so their
criteria stayed outside the closed rule and evidence catalogs schema v2 enforces
— the check that stops a profile demanding a class no check may emit. And a log
line that claims work it did not do is worse than silence.

The migration now verifies that the file replacing the instance's is actually at
the target schema before saying so, and a test requires every shipped profile to
be current, so the copy branch's assumption is checked rather than trusted.

Found while adopting the release in a consumer repository, not from a report: the
log said migrated twice per run and the files never changed.
