---
spec: pose-delegated-review-dispatch
category: added
breaking: false
refs:
---

`pose review dispatch <bundle|scope> --via <adapter> --apply` hands a sealed review to a configured second agent — Codex, Claude Code or any command declared in `.pose/policy/reviewers.json` — on a disposable worktree at the sealed commit, with the brief on stdin, and records the run under `.pose/review-runs/`. A run that changes files, times out, exceeds its budget or fails is recorded as failed, never as a review, and dispatch never records an attestation.
