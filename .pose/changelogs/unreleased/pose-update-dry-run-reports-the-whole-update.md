---
spec: pose-update-dry-run-reports-the-whole-update
category: fixed
breaking: false
refs:
---

`pose update --dry-run` previews the running engine's whole instance update, including file changes, engine stamps and configuration requests, without applying changes. Incomplete evaluation is reported explicitly, and `doctor` and `check` warn when an instance was last updated by an older engine.
