---
spec: pose-dependency-digest-pinning
category: security
breaking: false
refs:
---

Every GitHub Action and both container base images are now pinned to immutable
digests, with the version they correspond to recorded beside each one. The
GitHub-owned actions had been tag-pinned under an owned exception and the
container bases carried no digest at all, which together were the largest open
block in the OpenSSF baseline. A moving tag meant a release build executed
whatever that tag pointed at that morning, which is not something the published
provenance could describe.
