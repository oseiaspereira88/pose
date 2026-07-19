---
spec: pose-harne8-control-plane-integration
category: added
breaking: false
refs:
---

Ratifies how POSE composes with Conductor, Harness, GraphForge and
Portal (POSE governs, Conductor orchestrates, Harness executes,
GraphForge contextualizes, Portal presents) and closes the one missing
link: new `pose reconcile-evidence` records a Harness execution result
as identity-bound, append-only evidence — a second record for an
already-reconciled request is rejected unless explicitly superseded, and
even then the prior record is never edited or removed. Documents the
existing idempotent run-submission and policy-filtered projection
contracts as the durable-state and Portal/GraphForge consumption points.
Offline degradation (every open-core workflow completing with zero
Harne8 configuration) is now proven by an executable test, not just
documented.
