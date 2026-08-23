---
spec: pose-review-bundle-scope-isolated-listing
category: fixed
breaking: false
---

Fixed scope-isolated listing for review bundles (`ListReviewBundles`), attestations (`ListReviewAttestations`), and review attempts (`ListReviewAttempts`). Scoped queries inspect headers before strict integrity loading, ensuring that an invalid or corrupted review artifact belonging to an unrelated scope does not abort review verification or closeout for other specs.
