---
spec: pose-extension-reference-publication
category: fixed
breaking: false
refs:
---

The v0.20.0 release failed to publish: the step that signs the reference
extension writes the package and its Sigstore bundle to the repository root, and
goreleaser refuses to run against a dirty working tree. Both artifacts are now
ignored, exactly as the compatibility report already was.
