---
spec: pose-manual-distribution-merge
category: fixed
breaking: false
refs:
---

`pose upgrade` now delivers POSE.md and AGENTS.md to repositories that already
have them, refreshing the engine's own sections while preserving everything the
repository wrote under its instance-owned sections. Until now the manuals only
reached fresh installs: a plain upgrade skipped them and `--force` overwrote
local content wholesale.
