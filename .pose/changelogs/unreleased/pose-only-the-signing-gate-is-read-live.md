---
spec: pose-only-the-signing-gate-is-read-live
category: added
breaking: false
refs:
---

A test fails if a second policy field is read from the live policy while judging
a sealed bundle, or if the signing requirement stops being read that way.

Everything a sealed bundle is judged by is sealed with it, so a setting flipped
today cannot re-judge a review recorded years ago.
`require_signed_attestations` is the one deliberate exception: it is a bar rather
than a permission, and a bundle sealed before a project started requiring
signatures must not be permanently exempt from it.

That exception was one line of code and one paragraph of ADR. Sealing it by
symmetry with its neighbours would look like tidying and would quietly exempt
every existing bundle from a security requirement; adding a second live read
would reopen the retroactive judgement three amendments have closed. Both are
changes that pass review by looking consistent, so the count is asserted rather
than remembered.
