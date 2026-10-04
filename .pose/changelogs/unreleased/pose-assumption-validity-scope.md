---
spec: pose-assumption-validity-scope
category: added
breaking: false
refs:
---

A material assumption can declare `Valid scope:` and pinned `Stale trigger:` entries (`doc:<path>@<sha256 prefix>` and other local kinds). When the pinned content changes, the obligation projection emits a `premise-stale` judgment obligation, advisory on review and scoped to the assumption and the decisions based on it; the recorded evidence stays, described as judged against the old content. Calendar expiry is not evaluated, trivial assumptions need no field, and existing design digests are unchanged.
