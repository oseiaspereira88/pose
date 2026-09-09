---
spec: pose-release-boundary-rehearsal
category: fixed
breaking: false
refs:
---

The two sequences that only a real release had ever run now run in tests.

`pose update` is exercised end to end against a local release server: a binary
built at an older version fetches the release metadata, downloads the archive,
replaces itself on disk and hands off to what it wrote. Reintroducing the
`os.Executable()` call that failed the v2.0.0 release reproduces that failure
locally, which nothing could do before — reaching the handoff used to require an
actual download.

The review policy this engine writes is loaded by the binary of the previous
release, built from its own tag. The probe is `review-plan` rather than `check
--strict`, because `check` reads the policy through a small struct of its own
and never reaches the loader that enforces the schema; that wrong probe is what
made the v2.0.2 compatibility claim look true when it was not.

The release endpoints became package variables so a test can point a build at a
local server with `-ldflags -X`. They are deliberately not read from the
environment: where `pose update` fetches its own replacement from is not
something a shell variable should decide.
