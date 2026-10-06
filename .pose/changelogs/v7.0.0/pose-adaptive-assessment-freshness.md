---
spec: pose-adaptive-assessment-freshness
category: changed
breaking: false
refs:
---

Component assessments now record what they were computed from — the committed tree of the component, the engine version and the validation matrix — and `pose assess discover --if-stale` reuses an assessment whose inputs are unchanged and refreshes one that is stale, saying why. Age alone never stales an assessment. The feature and closeout workflows and the AGENTS template use `--if-stale` instead of rescanning every component before and after each change, and a missing or stale assessment of a component an in-progress spec declares appears in Attention as advisory work.
