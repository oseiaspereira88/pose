---
spec: pose-component-aware-review-plans
category: added
breaking: false
refs: ISSUE#13
---

Review can now resolve a deterministic effective plan for the actual components
affected by a spec, milestone or roadmap. The plan composes typed review
overlays, component-specific criteria and safe native POSE tool recommendations,
and binds new review attempts to both scope and plan digests.

The migration also preserves the configured exemption for completed scopes and
approved attempts that predate review-policy or component-aware adoption.
