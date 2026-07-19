---
spec: pose-release-compatibility-matrix
category: added
breaking: false
refs:
---

Each release now proves its parts fit together: a versioned
`compatibility.json` declares the supported engine, schema and upgrade pairs,
and a release gate re-runs the version, catalog, install and scaffold
contracts against the candidate, exercises every supported prior-version
upgrade from checksum-verified artifacts and publishes the resulting
compatibility report with the release.
