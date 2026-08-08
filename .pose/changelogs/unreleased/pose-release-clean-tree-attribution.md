---
spec: pose-release-clean-tree-attribution
category: changed
breaking: false
refs:
---

The release pipeline now asserts a clean worktree after each step that could
write, failing immediately and naming the step. goreleaser refuses to build from
a dirty tree but is the last thing that would notice, so the failure used to
arrive after the tests, the compatibility matrix, the security gate and the
extension signing had all run — listing files rather than the step that wrote
them. Legitimate build outputs are declared at the call site.
