# ADR: Review evidence versus delivery-target presence

## Status
Accepted

## Context
`reviewScopeRequiresValidationEvidence` correctly requires evidence for a spec with implementation components, even when it declares no delivery target. Attestation coverage reused that answer as `hasDeliveryTarget`. Consequently, a `validate` tool gated by `delivery-target-declared` cannot be deferred for a componentful spec without a target. The adopter reproduced this failure after its harness checks passed.

## Decision
Keep the evidence requirement and add a separate lookup for actual delivery declarations. Use the latter only for tool preconditions and the all-criteria-inapplicable gate. A root module target remains a delivery declaration, but its `.` path is not a component directory to map. Classify `.mcp.json` as an exact governed manifest.

## Consequences
Componentful specs still require current test evidence. Tools conditioned on a declared delivery target follow the target inventory instead of inferred component presence. Invalid paths and unclassified subjects retain their blockers. The change is covered by positive and negative tests under `pose-review-root-binding-parity`.
