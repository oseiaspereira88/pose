---
spec: pose-installer-local-binary-precedence
category: fixed
breaking: false
refs:
---

`install.sh` now installs from the POSE binary shipped beside it instead of
downloading the previously published release first, so a release bundle installs
offline and an install is never validated by an older engine than the one it
delivers. The installer also runs `pose upgrade` and `pose check --strict`
correctly — both were being passed a directory they do not accept, which printed
a usage banner and silently skipped the migration and the final gate.
