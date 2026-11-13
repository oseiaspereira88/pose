---
spec: pose-claude-reviewer-runs-tests-in-sandbox
category: added
breaking: false
---

- Optional per-agent `review_profile` selects fresh and resumed governed-review commands without widening read tasks, adjudication or smoke.
- This repository's Claude review profile executes tests in the mandatory shell sandbox, with private Go cache/temp scratch and absolute write denials for sealed source and the shared Git common directory. Missing bubblewrap, socat or jq refuses launch with installation advice; sandbox failure never falls back to an unrestricted command.
