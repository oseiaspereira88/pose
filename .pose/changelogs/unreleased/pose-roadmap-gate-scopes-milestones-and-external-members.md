---
spec: pose-roadmap-gate-scopes-milestones-and-external-members
category: fixed
breaking: false
refs:
---

`pose close milestone:<roadmap>/<id>` now gates on that milestone alone: its federated acceptance and its own members, without the roadmap's cut criteria. Before, it ran the whole `roadmap-check --strict`, so an open spec in a later milestone refused the close of an earlier one. `roadmap-check` no longer looks up members owned by another project in local closeout, which reported every `xref:` member as not terminal; those members stay gated by federated acceptance.
