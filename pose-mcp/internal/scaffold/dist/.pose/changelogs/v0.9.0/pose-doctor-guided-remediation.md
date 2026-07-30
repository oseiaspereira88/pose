---
spec: pose-doctor-guided-remediation
category: added
breaking: false
refs:
---

`pose doctor` findings now carry a stable code, evidence and a
`remediation_class` (`fixable`/`detectable`/`blocked`) alongside the
existing level/message/hint, versioned by a new `doctor_schema_version`
field — every prior JSON field is unchanged. `pose doctor --fix` previews
confined, reversible repairs (missing pre-commit hook, stale `.mcp.json`,
a broken `.claude/skills` symlink) without mutating anything; `--fix --yes`
applies them and immediately rechecks, reporting per-finding success, and
is idempotent on repeat runs. `--only <check>` scopes a fix to one finding.
Doctor output is defensively redacted against secret-shaped content.
