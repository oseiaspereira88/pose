---
spec: pose-update-reports-what-it-delivered
category: fixed
breaking: false
refs:
---

`pose update --force` and `pose install` no longer stop at the index step when
the instance's own state is invalid. A corrupt changelog fragment used to end a
run that had already delivered every machinery tree with "scaffold refresh
failed", without saying what had been written or that the fragment was there
before the run.

The run now reaches the final gate, which reports the failure, whether it
predates this run, and that the delivered files are not rolled back. The exit
code is still non-zero — the instance fails its strict gate — but the message is
about the instance, not about a refresh that succeeded.
