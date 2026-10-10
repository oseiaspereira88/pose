---
spec: pose-update-seeds-an-answerable-maintainer
category: fixed
breaking: false
refs:
---

When `pose update` opens configuration-review requests to the maintainer role and nobody holds it, the update now warns and names `pose setup`; answering such a request says that nobody holds the role and how to name its holder, instead of only rejecting the actor.
