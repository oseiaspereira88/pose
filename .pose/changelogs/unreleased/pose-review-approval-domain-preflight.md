---
spec: pose-review-approval-domain-preflight
category: security
breaking: false
---

The shared review-attestation writer now refuses an approving attestation the
verifier would reject before writing anything, with one typed refusal for the
Store, `pose review attest` (preview, apply, sign and envelope import),
`pose review auto-attest` and the MCP `pose_review_attest` tool. It also refuses
the approval of a superseded or stale bundle. A trusted signature over invalid
criteria still verifies identity but is never recorded as an approval;
legitimate negative decisions remain recordable, and persisted history is not
rewritten. Classified for a minor release.
