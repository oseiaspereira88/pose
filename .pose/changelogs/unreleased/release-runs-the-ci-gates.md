---
spec: release-runs-the-ci-gates
category: fixed
breaking: false
---

The Portuguese README pins the released version again: in 6.0.0 its checksum-pinned install instructions downloaded 5.0.8. The release workflow now runs the whole CI workflow at the commit it publishes and waits for it to pass. 6.0.0 was cut from a commit whose CI failed on two gates that the release workflow did not run.
