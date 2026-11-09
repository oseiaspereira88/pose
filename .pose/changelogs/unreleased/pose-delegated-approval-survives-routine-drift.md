---
spec: pose-delegated-approval-survives-routine-drift
category: fixed
breaking: false
---
A scope that is done keeps the approval a delegated review gave it when the repository changes afterwards, as it already kept an approval a person attested. Specs closed with a delegated review no longer read "no review attempt exists" once another spec changes a file they sealed (`validation-matrix.json`, `POSE.md`). The approval is verified as sealed: the signature under a pinned issuer, the exact run record, transcript and bundle it binds; revoking the issuer's pin, altering the record or reopening the scope still ends it, and an open scope keeps the live checks.
