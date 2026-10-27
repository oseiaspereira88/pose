---
spec: pose-delegated-review-conclusions
category: added
breaking: false
---

Add `pose review conclude <run>`: the typed conclusion of a completed delegated review run (the pose-draft block's `criteria` and `tools`, asked for by brief template `review-brief/3`) is recorded as its signed, run-backed attestation by the native issuer named with `--issuer` or `conclusions.issuer` in `delegation.json`, through the same domain writer and preflight as `pose review attest`. A rejecting conclusion is recorded too; a run that fails a check records no attestation and its conclusion record in the `review-conclusions` directory of `.pose` keeps the reasons; a run concludes once. With `delegated_review: true` and `conclusions.issuer`, `pose delegate review` concludes the run as it completes.
