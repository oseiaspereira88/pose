---
spec: pose-doctor-reports-a-profile-left-behind
category: added
breaking: false
refs:
---

`pose doctor` reports any review profile in the instance below the schema the
engine enforces, naming it and the command that migrates it.

Two profiles shipped at `schema_version: 1` for long enough that every instance
installed in that window still carries them, and the migration that should have
moved them reported success without doing so. The shipped files are current now,
so the next `pose update` migrates them — but until it runs, those profiles are
exempt from the closed rule and evidence catalogs schema v2 enforces.

That exemption is the substance rather than the version number: a v1 profile
skips the check that stops a profile demanding an evidence class no registered
check may emit, so the instance can plan a gate only a fabricated disposition
could pass. An instance with no profiles produces no finding.
