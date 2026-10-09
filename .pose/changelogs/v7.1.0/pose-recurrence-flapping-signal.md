---
spec: pose-recurrence-flapping-signal
category: added
breaking: false
refs:
---

`pose recurrence-check` now reports a task whose runs keep switching between failing and passing as a `flapping` warning (default: 4 transitions in the window, `--flap-threshold N`), counted in `recurrence.flapping_keys`. It never changes the exit code.
