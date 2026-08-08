---
spec: pose-package-channel-delivery
category: fixed
breaking: false
refs:
---

The Homebrew formula and WinGet manifests are now published as release assets,
so the install command in the documentation resolves. It pointed at
`releases/download/vX.Y.Z/pose.rb`, which had never been published — the
manifests were only retained as a CI artifact.

The clean-host verification for those channels also runs for the first time. It
triggered on `release: published`, which never fires for a release created by
the workflow's own token, and it invoked the Go module from the repository root
instead of `pose-mcp/`. Both are fixed.
