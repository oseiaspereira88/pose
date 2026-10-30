---
spec: pose-review-subject-later-removal
category: changed
breaking: false
---

- A milestone review bundle now seals after its specs closed and a later spec removed their files: a review subject path that a later spec removed reads as that removal (the removing spec and deleting commit the delivery graph accepted), and a done member spec whose later commits changed its preparation contributes the approved review the closeout gate already retains. An absent path without an accepted removal, a member that is not done, or a member whose latest review is negative still blocks the seal.
