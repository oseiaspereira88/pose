---
spec: pose-governance-outcomes-v2
category: changed
breaking: true
refs:
---

`pose stats governance` and `pose_governance_stats` now emit schema 2 with replay-safe report identity, bounded band/type cohorts, source provenance, and allowlisted finding-to-remediation references. The projection remains local and read-only; schema 1 consumers must be upgraded before adopting this contract.
