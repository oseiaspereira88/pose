---
spec: pose-material-equivalence-reuse
category: added
breaking: false
refs:
---

When a review bundle is superseded, `pose review verify` now explains reuse criterion by criterion: `equivalent` when every input the answer depends on is unchanged, `changed` with the exact inputs that differ (subject paths, evidence, governing inputs, tools, definition), or `new`; each reused answer names its origin attestation. Reuse itself still follows the criterion input digest and the policy, so a bookkeeping edit keeps answers valid and a code change invalidates the ones that depend on it.
