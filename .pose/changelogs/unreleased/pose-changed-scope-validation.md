---
spec: pose-changed-scope-validation
category: added
breaking: false
refs:
---

`pose validate --changed-from <rev> [--changed-to <rev>]` selects the minimum
safe module set from Git changes, declared dependency edges and policy —
with `--explain` printing every decision and unselected checks recorded as
skipped with a machine-readable reason. A root-level or unmapped change runs
everything; without the flags, full validation is unchanged.
