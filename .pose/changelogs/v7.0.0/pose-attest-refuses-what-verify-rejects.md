---
spec: pose-attest-refuses-what-verify-rejects
category: changed
breaking: false
refs:
---

`pose review attest` (preview and `--apply`) and `pose review auto-attest --apply` now run the verifier over an approving attestation before it is written and refuse it, listing every reason, when `pose review verify` would reject it — an unmapped structural fact, an undeclared basis, a mapping without a reason, evidence absent from the bundle. Negative decisions are still recorded as audit.
