---
spec: pose-delivery-target-root-module-evidence-matching
category: fixed
breaking: false
---

Fixed delivery target module validation and validation evidence attribution in single-module projects. Allowed declaring `module:.` without triggering root escaping errors, updated `moduleMatchesTarget` and `reviewBundleEvidence` to attribute root validation checks (`Module: "."`) to declared delivery targets, and formally added the Continuous Opportunity Scouting requirement under POSE Contributor Mode.
