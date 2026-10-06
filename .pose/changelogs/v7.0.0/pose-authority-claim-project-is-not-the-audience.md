---
spec: pose-authority-claim-project-is-not-the-audience
category: changed
breaking: true
refs:
---

A signed authority claim binds its project and its audience separately: `project` is compared with the new `authority_project` in the review policy (the project the decision governs) and `audience` with `authority_audience` (the verifier installation). Verified identity assurance now requires `authority_project`, for review and action-request claims alike, so a claim issued for one project served by a shared verifier no longer satisfies another. Breaking only for a policy that already adopted `verified`.
