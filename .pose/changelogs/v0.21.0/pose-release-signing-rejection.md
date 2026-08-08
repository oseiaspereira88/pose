---
spec: pose-release-signing-rejection
category: changed
breaking: false
refs:
---

The artifact-identity gate's rejection path is now exercised on every CI run.
It had always contained the logic — an unsigned artifact, a missing SBOM or an
identity mismatch fails the release — but no run had ever triggered it, so the
behaviour was asserted rather than demonstrated. Four deliberately broken
artifact sets now prove it.
