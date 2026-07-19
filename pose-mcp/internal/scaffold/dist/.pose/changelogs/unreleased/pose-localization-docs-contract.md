---
spec: pose-localization-docs-contract
category: fixed
breaking: false
refs:
---

Fixed a real locale-parity gap: the default (English) `knowledge` and
`doc-audit-report` templates were actually written in Portuguese with no
`pt-BR` translation on file, so a `pt-BR` install silently got them
"right" by accident while an English install got them wrong. The locale
overlay now uses one consistent path convention for every overlaid
directory (templates, workflows, rules, skills), both languages are
correct, and the parity check that should have caught this now covers
templates too. Every documented `pose <command>` across the README and
docs site is now tested against the CLI's own dispatch table, every docs
page carries a visible Diátaxis type + version-applicability line, and
docs are scanned for secret-shaped and unsafe-command patterns.
