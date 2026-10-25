---
spec: pose-delegated-review-run-continuity
category: added
breaking: true
---
Continue interrupted delegated reviews with the original session, sealed inputs and attempt quota. Declare adapter limit/session/resume behavior, inspect partial drafts and reset times, wait once for a bounded reset, and choose any cross-adapter fallback explicitly. Timeouts and observed process loss now produce interrupted records; legacy stale records remain readable. Continuations are counted separately from attempts and cannot apply a review attestation.
