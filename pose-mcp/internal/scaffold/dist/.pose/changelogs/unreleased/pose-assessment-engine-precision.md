---
spec: pose-assessment-engine-precision
category: fixed
breaking: true
refs:
---

The assessment engines no longer report more than they observed. Technical debt
counts as covered only when a document cites the marker's file or id, not merely
its component; `pose assess discover --component` keeps components it did not
scan in the consolidated view instead of erasing them; and integration gap ids
derive from the contract's identity, so adding a contract no longer renumbers
every later gap. Expect previously covered debt to reappear as uncovered and
`gap_id` values to change shape once.
