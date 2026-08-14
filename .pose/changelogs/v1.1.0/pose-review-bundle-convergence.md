---
spec: pose-review-bundle-convergence
category: changed
---

Add opt-in sealed review bundles and separate attestations so POSE closeout
converges without self-invalidating rereview. The release includes scoped
delivery provenance, supersession deltas, targeted criterion reuse, an offline
CLI flow and a read-only MCP projection.

The release closeout also preserves archived-fragment provenance, excludes
agent-local `.qwen` worktrees from discovery, reconciles recorded release
change sets during artifact checks, and returns a filesystem error instead of
terminating the process when the embedded distribution is invalid.
