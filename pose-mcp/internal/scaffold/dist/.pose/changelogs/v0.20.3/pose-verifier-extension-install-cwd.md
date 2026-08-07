---
spec: pose-verifier-extension-install-cwd
category: fixed
breaking: false
refs:
---

The independent verification's extension-install check now installs into the
instance it created. `pose extension install` acts on the current directory
rather than on a target argument, so without a `cd` the check installed
somewhere else and then asserted the rule was missing. Signature verification
and tamper rejection were already passing against the published artifact; this
was the remaining leg.
