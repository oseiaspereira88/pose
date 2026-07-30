---
spec: pose-safe-validate-orchestration
category: added
breaking: false
refs:
---

Agents can now request validation through MCP without unsafe local
execution: `pose_validate_request` resolves a digest-pinned plan,
`pose_validate_approve` requires a bound Execution Identity and the exact
plan digest (substitution is rejected), and `pose_validate_submit` hands the
approved plan to a pluggable Harness executor — idempotently, and only when
one is configured. Local `pose validate` is completely unaffected.
