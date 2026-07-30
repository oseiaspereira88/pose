---
spec: pose-structured-validation-results
category: added
breaking: false
refs:
---

`pose validate` now emits a versioned structured result — `--json`, `--junit`
and `--sarif` — from one canonical model: stable check IDs, command metadata,
timing, severity, distinguishable outcomes (infrastructure errors never
masquerade as check failures), deterministic skip reasons, bounded captured
output and secret redaction. Text output is unchanged; machine formats are
additive and CI/scanner-consumable.
