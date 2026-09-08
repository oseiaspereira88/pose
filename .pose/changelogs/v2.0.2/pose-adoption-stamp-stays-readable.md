---
spec: pose-adoption-stamp-stays-readable
category: fixed
breaking: false
refs:
---

`pose update` writes a contract's adoption date to the top-level key that held
it before `contract_adoptions` existed, and only uses the map for a contract
that has no such key. The review policy decoder refused unknown fields, so a
policy carrying `contract_adoptions` was rejected outright by any engine older
than 2.0.0 — not ignored, rejected:

```
pose: invalid schema-v2 review policy: json: unknown field "contract_adoptions"
```

A release meant to let an instance adopt new rules without losing its history
made that instance unreadable to the engine it was adopting from. Every
environment not yet updated was affected: a developer's machine, a pinned CI
job, a container image built last week.

The decoder now ignores fields it does not know, so the next field added does
not repeat this.
