---
spec: pose-help-names-flags-the-parser-accepts
category: fixed
breaking: false
refs:
---

Five `--help` screens named flags their commands refuse. `pose update` offered
`--schema-only` and omitted `--no-self` and `--locale`; `pose dora-metrics`
offered `--app`, `--env` and `--since-days` for `--application`, `--environment`
and `--window-days` (default 30 days, not 90); `pose adoption-metrics` offered
`--since-days` and `pose state` offered `--json`, neither accepted; and
`pose release record` showed `--provider-state` where the command takes
`--event` and `--evidence`. Each help entry now matches the command.
