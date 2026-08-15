---
spec: pose-install-gate-failure-recovery-notice
category: fixed
breaking: false
refs: ISSUE#18
---

When `pose install`'s final `--strict` gate fails, the error message now explicitly says every file was already written before the failure (the command does not roll back) and names both recovery paths: the `.pose-backup` copy kept for every overwritten existing file, and `git status`/`git diff` on the target.
