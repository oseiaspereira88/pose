---
spec: pose-review-plan-producible-evidence-classes
category: fixed
breaking: false
refs:
---

A manual review attestation can now be recorded for a scope with mapped
components. The planner used to synthesise a per-component `validate` tool
demanding evidence of class `validation`, which no registered check is allowed
to emit, so `pose review attest` refused every truthful disposition and
`pose review auto-attest` was the only path that completed. Tool evidence
classes now come from the governing profile, and any class a check cannot emit
is dropped with a warning naming it.
