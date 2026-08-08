---
spec: pose-actions-node24-bump
category: changed
breaking: false
refs:
---

The release and CI workflows now run their first-party actions on Node.js 24.
They had been declaring the deprecated Node.js 20 and passing only because the
runner substituted a newer runtime and annotated every run — an override GitHub
controls and will withdraw, at which point every workflow would have failed at
once. Third-party actions remain pinned to full commit SHAs.
