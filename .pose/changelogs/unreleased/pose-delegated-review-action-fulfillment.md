---
spec: pose-delegated-review-action-fulfillment
category: added
breaking: false
---

Add delegated-review action requests. `pose review request <scope> --kind delegated-review` opens a request that binds the bundle, the brief, the notes, the policies, the accepted agents and the destination of the conclusion; no actor answers it. A local `pose delegate review --request`, `pose action fulfil` (flags or `--envelope`) from a CI job, and the MCP tool `pose_action_fulfil` for Harne8's Conductor submit the same envelope, and the engine records the fulfilment only for a completed run of that brief whose signed proof, attempt ledger, containment and recorded conclusion verify — again on every read. A rejecting run fulfils without approving; a superseded bundle or changed policy invalidates the request; a person is asked by exception (a plan reserving the review for a person, a spent budget, reviewers in conflict, a business decision in the draft). Run proofs now verify in any checkout of the project: the origin worktree is recorded, not required of the reader.
