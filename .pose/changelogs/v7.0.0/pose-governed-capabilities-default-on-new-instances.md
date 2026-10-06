---
spec: pose-governed-capabilities-default-on-new-instances
category: added
breaking: false
refs:
---

A new instance adopts the governed capabilities: when `pose install` creates the review policy it turns on agency readiness, contract nodes, atomic start and causality closeout with the `structural-materiality@1` overlay, dated with the install day. Existing instances are unchanged — install over an existing policy and `pose update` never adopt — and toggle each capability with `pose adopt <capability> [--off] [--date YYYY-MM-DD] [--apply]`, which writes only a policy the reader accepts. `state --governance` names the command for a capability not adopted, and `pose doctor` warns when agency readiness has no principal holding a role.
