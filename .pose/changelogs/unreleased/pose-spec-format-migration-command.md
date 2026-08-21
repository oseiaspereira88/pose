---
spec: pose-spec-format-migration-command
category: added
breaking: false
---

Added the `pose spec-format` CLI command suite with `migrate <slug>|--all` and `status` subcommands, allowing automated migration of legacy specifications to modern date-prefixed formats (`YYYY-MM-DD-<slug>/spec.md` or `YYYY-MM-DD-<slug>.md`) while strictly preserving directory envelopes for specs with companion artifacts (`amendments.jsonl`).
