---
spec: pose-governance-gate-activation
category: fixed
---

Harden the 1.1 release closeout by preserving archived-fragment provenance,
excluding agent-local `.qwen` worktrees from discovery, reconciling recorded
release change sets during artifact checks, and returning a filesystem error
instead of terminating the process when the embedded distribution is invalid.
