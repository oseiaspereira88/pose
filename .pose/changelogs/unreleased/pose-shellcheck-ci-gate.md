---
spec: pose-shellcheck-ci-gate
category: changed
breaking: false
refs:
---

CI now runs shellcheck over every shipped or executed shell script, including
`tests/release/` and `scripts/`, and fails on findings at warning severity. It
previously covered only `install.sh` and the installer E2E — leaving out exactly
the scripts that run during a release, where three consecutive cuts failed on
unbound-variable defects that this check reports.
