---
spec: pose-fragment-error-clarity
category: fixed
breaking: false
refs:
---

A malformed changelog fragment now says which file and which field. It reported
only `malformed release fragment <name>`, leaving an operator to guess between a
missing `spec:`, an invalid `category:` and an empty body — in a file the message
did not locate. It surfaced during `pose upgrade`, where the failure aborts the
scaffold refresh, so the least useful moment to be given a riddle.
