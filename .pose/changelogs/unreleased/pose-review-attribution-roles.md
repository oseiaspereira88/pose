---
spec: pose-review-attribution-roles
category: added
breaking: false
refs:
---

Review attestations can record who prepared, concluded, confirmed and applied a review, with `--prepared-by`, `--concluded-by`, `--confirmed-by` plus `--confirmation-mode adopted-conclusions|authorized-operation`, and `--applied-by`. Roles are never filled from `--reviewer`, a confirmation is bound to the exact content written, and review surfaces render the difference between adopting conclusions and authorizing an operation. Older records show as `legacy-undifferentiated`, and `pose review attribution-supplement` clarifies them by reference without altering them; the sixteen 6.3.0 cycle attestations now carry such supplements.
