---
spec: pose-review-subject-unclassified-removals
category: fixed
breaking: false
refs:
---

Deleting a file the review subject classifier does not recognise no longer
blocks the whole bundle. The removal is classified and stays visible in the
subject, so a reviewer still sees it; a created or modified path the classifier
does not recognise keeps failing closed, since it still exists and can be
inspected.

`removed` is a new public subject class, and it counts toward the input digest
of every criterion — including those that would otherwise be reusable — so a
deletion invalidates criterion reuse rather than being covered by a verdict
issued before it. The sealing clause of the sealed-review-bundles ADR is amended
accordingly.
