---
spec: pose-public-claims-publication-provenance
category: changed
breaking: false
refs:
---

`pose public-claims` no longer calls the locally read engine version "released". It reports the candidate version and its release lifecycle state (unprepared, prepared, tagged, published, verified) and, separately, the latest version whose publication is proven by retained release events, or `unproven`. The JSON keeps `released_version` with its old value for one minor, marked deprecated, and adds `version_provenance`.
