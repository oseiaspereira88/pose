---
spec: pose-update-configuration-review
category: added
breaking: false
refs:
---

`pose update` now asks the maintainer about capabilities the project has not decided for the engine it runs: it writes `.pose/specs/<date>-pose-configuration-review-<version>.md` and opens one decision request per capability (adopt, decline or defer, with the recommendation), addressed to the maintainer role. Nothing is adopted by the update; a second update for the same version asks nothing again. `pose adopt --request <act-id> --apply` applies an answered request — recording the answer's reason and the request id for a decline or deferral — and `pose setup` lists open and answered requests and can answer and apply them at a terminal.
