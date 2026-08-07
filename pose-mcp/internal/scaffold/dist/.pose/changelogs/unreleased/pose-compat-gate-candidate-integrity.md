---
spec: pose-compat-gate-candidate-integrity
category: fixed
breaking: false
refs:
---

`pose upgrade` no longer discards manual text a repository wrote outside a
section of its own: when refreshing POSE.md or AGENTS.md would drop such a line,
the pre-merge file is kept as `<doc>.pose-backup` and the installer says so.
The release compatibility gate also stopped replacing the candidate binary with
the last published release while running, which had left most of its upgrade
pairs validating the previous version instead of the one being cut. The
supported-upgrade window now starts at 0.18.1; earlier releases stay published
but are no longer exercised.
