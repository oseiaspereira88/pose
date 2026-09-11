---
spec: pose-manual-merge-backs-up-only-local-edits
category: fixed
breaking: false
refs:
---

The POSE.md/AGENTS.md merge backs up only what an instance edited. It counted
any line of the local manual missing from the merged result as lost, so every
release that reworded one of its own sections backed the manual up with
"content outside instance-owned sections was not preserved" — the same false
positive 5.0.4 removed from machinery delivery. The delivery manifest now
records what POSE wrote to each section; a section still matching it is simply
replaced, and a note the instance wrote inside an engine-owned section is still
backed up. A manual with no record yet is backed up once, saying why, and
recorded.
