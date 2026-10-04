---
spec: pose-effective-governance-projection
category: added
breaking: false
refs:
---

`pose state --governance [--scope <ref>] [--json]` shows, live and read-only, what governance is in force in this instance rather than what the engine ships: for each review contract, optional capability and gate, whether it is supported, configured, applicable and effective, with reason codes such as `not-adopted`, `no-readiness-cutoff` or `legacy-cutoff:<date>`. With `--scope`, it compares the contracts the scope's newest sealed bundle stamped with what the policy seals today. `pose_project_state` carries the same projection as `effective_governance`.
