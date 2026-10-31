---
spec: pose-review-approval-follows-children
category: changed
breaking: false
---
A milestone or roadmap stops verifying approved while the review of a child no longer stands: rejected, with lost attestation history, stale, or reopened under a done parent. It recovers with the child, with no reseal. Bundle selection honours replacement lineage everywhere, so retention no longer returns a bundle its replacement superseded, whatever the seal times say.
