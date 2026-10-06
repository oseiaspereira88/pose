---
spec: pose-governance-wait-rework-observability
category: added
breaking: false
refs:
---

`pose stats governance --waits --rework` adds two dimensions beside the outcomes: request age, attributed waiting and known whole-spec blocking (simultaneous intervals counted once, nothing read as inactivity), and why review work was redone (what changed between superseded bundles, review disagreement or invalid earlier attestations, otherwise unknown). `scripts/bench-governance.sh` times state, check and index on one revision with corpus and engine version recorded. Bundle diffs no longer compute the per-criterion reuse explanation outside verify, and closeout reuses the bundle and attestation it already settled on to describe review assurance.
