---
spec: pose-composition-contract
category: added
breaking: false
refs:
---

`composition-contract.json` publishes how this repository's images are built and
configured — build context, Dockerfile, port and the full environment surface of
each service — so a consumer composing them can derive those facts instead of
restating them. The environment names are derived from the source rather than
listed by hand, including the keys built by prefix concatenation that appear
nowhere as literals; a check fails when the published contract and the
repository disagree, which is the silent failure a rename used to produce.
