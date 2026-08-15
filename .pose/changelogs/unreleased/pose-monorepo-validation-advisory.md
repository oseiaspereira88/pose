---
spec: pose-monorepo-validation-advisory
category: added
breaking: false
refs: ISSUE#23
---

`pose doctor` now recognizes when a monorepo's root validation script delegates to workspace members (e.g. `npm test --workspaces`) while POSE also validates those members directly, and recommends the existing `moduleOverrides.<path>.replaceDefaultChecks` fix by name. `pose validate` also gained `--root-only` and `--workspace <name>` as documented shortcuts for the existing `--module <path>` selector.
