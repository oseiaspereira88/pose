---
spec: pose-delegated-review-capability
category: added
breaking: false
---

Add `delegated-review` to the capability catalog. Fresh interactive `pose init`
offers it by default, detects supported agent CLIs without running them and
provisions native executor signing trust before activation. Refusal, headless
init or no supported CLI leaves it explicitly off without an agent policy or
key; existing instances are preserved and update only suggests adoption.
Existing-project adoption validates the exact signer, pin and project binding.
Explicit off allows new eligible plans to disclose same-actor execution as
`delegated-review-disabled`, while human and protected-baseline requirements
stay stronger, including signed run proof for the agent review of the
disabling change itself. Adopted review requires another vendor or model and escalates
when none is available. The B17 live run-proof barrier continues to apply to
older approvals without rewriting their records. Add the `pose-delegate`
skill and integrated rejection, remediation, automatic conclusion and closeout
journeys.
