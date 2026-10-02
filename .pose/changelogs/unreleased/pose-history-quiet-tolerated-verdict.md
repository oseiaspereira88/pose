---
spec: pose-history-quiet-tolerated-verdict
category: fixed
breaking: false
---

History checks in quiet mode emit one final verdict when unstaged or untracked
JSONL is tolerated, preserving strict failure and tolerant success exit codes.
