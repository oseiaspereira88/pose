---
spec: pose-delegated-verdict-destinations
category: added
breaking: false
---

`pose review conclude` (and the automatic conclusion of `pose delegate adjudicate|smoke` under `delegated_review` and `conclusions.issuer`) records the typed verdict of an adjudication or a smoke run where its origin reads it. An adjudication, briefed with `adjudication-brief/2` to answer each decision and requirement of the spec in `questions`, is recorded as an adjudication record of the spec that `lint-spec` reads: a question the current bundle's adjudication says does not hold fails the lint. A smoke run, briefed with `smoke-brief/2` to report `surfaces`, is recorded as validation results of the new evidence class `smoke`, one per delivery target, which `surface-check` reads. Records are verified on read against their run's proof and draft; a verdict that does not answer exactly what was asked records nothing.
