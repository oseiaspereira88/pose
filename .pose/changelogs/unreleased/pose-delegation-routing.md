---
spec: pose-delegation-routing
category: added
breaking: false
---

Route delegated runs by the implementation they judge. `pose review bundle --seal` records who implemented the subject, detected from the sealing agent's session or declared with `--lead <vendor>[/<model>]`, and a reseal of the same content keeps it. `pose delegate` without `--agent` prefers an agent of the opposite vendor and falls back through `routing.on_unavailable`, waiting for a known usage-limit reset within `routing.max_wait`; `pose delegate agents` shows each agent as available, limited, missing or with a failed probe. Every run records its routing decision and the independence it achieved, and `verify` and the closeout disclose it; a same-model run never satisfies `different-actor`. `provenance.implementation` in the policy is read only for bundles sealed before this change.
