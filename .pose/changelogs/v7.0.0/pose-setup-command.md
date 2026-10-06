---
spec: pose-setup-command
category: added
breaking: false
refs:
---

New `pose setup`: one map of the instance — project identity, principals and keys, the pre-commit gate, capabilities in force, capabilities new since the last configuration review, what needs setup, whether the configuration is committed — and one next step with its command. At a terminal it offers each open step (the hook, registering your key with the maintainer role, adopting, declining or deferring each new capability) and performs it only after an explicit yes; with `--json`, `--no-input` or no terminal it changes nothing. `.pose/policy/adoption-decisions.json` records `reviewed_version` (a fresh install records its own); `pose install` ends with `pose setup`, `pose update` names pending capability decisions instead of "Nothing to do", and `pose doctor` reports them as a next step.
