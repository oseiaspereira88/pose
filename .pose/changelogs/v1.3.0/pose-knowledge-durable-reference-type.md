---
spec: pose-knowledge-durable-reference-type
category: added
breaking: false
refs: ISSUE#25
---

`.pose/rules/knowledge-governance.md` and the `pose-knowledge` skill now document where a durable, non-architectural fact or convention belongs: `.pose/rules/`, not a `decision-log` forced into that shape for lack of a better option. `.pose/knowledge/`'s three types (`handoff`, `decision-log`, `note`) and their mandatory TTL are unchanged — this only adds guidance for telling a real decision apart from a durable fact that never expires.
