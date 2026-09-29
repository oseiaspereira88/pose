---
spec: pose-lint-spec-all-is-a-gate
category: changed
breaking: false
---

This repository's CI runs `pose lint-spec --all`, and it passes. Two done specs were brought under the template, one of them with a trace that cited a test that never existed, now replaced by a regression test for date-prefixed specs in `release prepare`.
