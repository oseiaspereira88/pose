---
spec: pose-release-version-source
category: added
breaking: false
---

Let a project declare the file that holds its own release version with `version_source` in `.pose/policy/release.json`, so `pose release plan`, `prepare` and `check` work outside the engine repository. A project with no declared source outside the engine repository is now refused with a message naming the field instead of being compared with the engine's version; policies that do not set it digest as before.
