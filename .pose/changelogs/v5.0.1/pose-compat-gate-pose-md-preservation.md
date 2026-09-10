---
spec: pose-compat-gate-pose-md-preservation
category: changed
breaking: false
refs:
---

The release compatibility gate customizes POSE.md as well as AGENTS.md in its
upgrade lab, and fails a supported upgrade that loses the customization.

The two manuals do not take the same path. A note appended to AGENTS.md stays in
it; the same note appended to POSE.md is dropped from the manual on upgrade and
kept in `POSE.md.pose-backup`. Only the first had ever been exercised against a
real prior release — the backup, the one thing that turns that removal into
something other than a silent loss, had not.
