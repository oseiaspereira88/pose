---
spec: retained-review-survives-an-invalid-newer-approval
category: fixed
breaking: false
---

A closed scope whose newest approval no longer validates, typically against an evidence rule introduced after it, keeps the older approval it was closed with again. 6.0.2 voided it, so a scope approved twice and never rejected read as unreviewed and `pose check --strict` failed. A newer rejection or request for changes still voids any older approval.
