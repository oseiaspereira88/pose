---
spec: pose-delegation-surface
category: added
breaking: false
---

Add `pose delegate`, the one surface for delegated executions: `review`, `adjudicate` and `smoke` over the delivered dispatch path, `ls`, `resume`, `cancel` and `agents`, with one vocabulary, `--dry-run` and versioned `--json` documents. `.pose/policy/delegation.json` (schema 2) replaces `reviewers.json`, which `pose update` converts deterministically. The child receives none of the parent agent's session variables, both vendors' streams normalize to one event schema bound to the run, and `pose review dispatch`, `resume` and `status` become deprecated aliases.
