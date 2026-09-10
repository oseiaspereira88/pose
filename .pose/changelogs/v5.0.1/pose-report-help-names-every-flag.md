---
spec: pose-report-help-names-every-flag
category: fixed
breaking: false
refs:
---

`pose report --help` names all sixteen flags `pose report` accepts. It named
four. Among the missing were `--validate-output`, which chooses the validation
log a report derives its outcome from and was documented nowhere, and
`--change-from`/`--change-to`, which record a spec's immutable change set. The
help also listed `--outcome` as `pass|fail|partial`; `skipped` and `unknown` are
accepted too.

A test now holds the help to the flags the parser accepts.
