---
spec: pose-recoverable-closeout-plan
category: added
breaking: false
refs:
---

`pose close spec:<slug> --plan|--apply|--resume` turns closeout into an ordered, recoverable plan computed from the repository: evidence is regenerated into `results_path` and indexed before sealing, the bundle is sealed, criteria answered by sealed evidence are recorded under the reviewer you name, and the guarded transition runs. The plan stops with exit 3 at a judgment criterion and never answers it; an interrupted run resumes without repeating steps or duplicating bundles and attestations, and a plan that no longer matches the repository is refused.
