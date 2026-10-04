---
spec: pose-v7-legacy-cleanup-plan
category: added
breaking: false
refs:
---

`pose migrate v7 --dry-run [--json]` inventories, without writing, what a future major would convert, keep or refuse — spec layouts, blocked specs, legacy and unknown review policy keys (an unknown key is reported as a gate that may be silently off), sealed bundles, attestations by identity mode and attribution, and incomplete transfers on this project's side — each with its removal criterion. `--apply` is refused in 6.x, and sealed records are never rewritten. The help catalog now lists `specs facts`, `followups --candidates`, `new-spec --surface`, `action list --present` and `stats governance --waits/--rework`.
