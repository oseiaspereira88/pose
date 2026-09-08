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
