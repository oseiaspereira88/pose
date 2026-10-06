---
spec: pose-install-and-upgrade-journeys
category: fixed
breaking: false
refs:
---

CI now walks a fresh install and an upgrade from the latest published release (verified against its checksums): both must pass the strict check and read clean in `pose doctor`, and the upgrade must open one review request per new capability, ask only once, and apply a signed answer through `pose adopt --request`. Walking them fixed two gaps: `pose setup` now asks for a maintainer whenever a request is addressed to a role nobody holds, and `pose doctor` names an available rule extension by the catalog id `pose extension install` resolves.
