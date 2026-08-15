---
spec: pose-install-locale-autodetect
category: fixed
breaking: false
refs: ISSUE#18
---

`pose install <target>` rerun on an already-localized instance, and `pose update --force` (which delegates to the same code path), no longer silently revert the project's machinery (`.pose/rules`, `.pose/templates`, `.pose/workflows`, `.agents/skills`, `AGENTS.md`, `POSE.md`) to English — the existing locale is now detected from the target's own `POSE.md`, the same way a plain `pose update` already did. An explicit `--locale` still always wins.
