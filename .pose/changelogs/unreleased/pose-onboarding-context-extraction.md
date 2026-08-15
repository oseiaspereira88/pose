---
spec: pose-onboarding-context-extraction
category: added
breaking: false
refs: ISSUE#21
---

`pose init`/`pose install` now excerpts a brownfield target's own `README.md`/`CLAUDE.md` into `AGENTS.md`'s "Project context" section on first install, instead of leaving the generic unfilled placeholder. The section is preserved on later `pose install`/`pose update` runs, so an extracted or hand-edited description is never silently reverted.
