---
spec: pose-flat-spec-amendments
category: fixed
breaking: false
refs:
---

`pose amend` now resolves the spec through the store, so flat dated specs (`.pose/specs/YYYY-MM-DD-<slug>.md`) record amendments like folder specs. Each flat spec gets its own `YYYY-MM-DD-<slug>.amendments.jsonl` beside it instead of the shared directory journal the sibling rule implied; lint, the MCP amendments view and `pose spec-format migrate` read and carry that journal, and existing folder journals are read unchanged.
