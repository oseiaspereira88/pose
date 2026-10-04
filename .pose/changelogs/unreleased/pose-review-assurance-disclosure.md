---
spec: pose-review-assurance-disclosure
category: changed
breaking: false
refs:
---

Review surfaces now say what a review record proves about its reviewer. `review-plan` prints the identity assurance in force; `review verify`, `review-check` and `closeout-check` print an assurance line and carry an `assurance` object that keeps required, declared and verified separation apart. A `human:` or `agent:independent-` prefix under declared assurance is shown as a declaration, a verified claim is shown with its principal, executions, issuer and audience, and cognitive independence is always reported as not observable.
