---
spec: pose-causality-closeout-adoption-cutoff
category: added
breaking: false
refs:
---

`causality_closeout_adopted_at: YYYY-MM-DD` stamps causality closeout only on scopes with a spec created on or after the date, and `overlay_adopted_at: {"<overlay>@<n>": "YYYY-MM-DD"}` selects a late-adopted overlay only for such scopes, so adopting either does not charge work already under way. A spec without a creation date is never exempted; malformed dates and dates for unlisted overlays are refused; effective governance states both.
