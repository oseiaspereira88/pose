---
spec: pose-governance-stats-local-freshness
category: fixed
breaking: false
refs:
---

`pose_governance_stats` now verifies bundle freshness through a local store, like `pose stats governance`. From pose-mcp it used to resolve qualified dependencies again for every attested bundle, which on a large history took more than ten minutes instead of seconds and could count freshness differently from the CLI.
