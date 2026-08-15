---
spec: pose-extension-catalog-resolution
category: added
breaking: false
refs:
---

`pose extension install <extension-id>` now resolves the ID against the latest published GitHub release's signed assets, downloads and safely extracts the package, and installs it through the same signature-verified pipeline a local directory install already used — no local directory required. `pose doctor`'s rule-extension recommendation can now be satisfied by a single runnable command for any extension published this way.
