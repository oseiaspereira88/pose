---
spec: check-strict-verdict-names-its-mode
category: fixed
breaking: false
---

`pose check` now names the mode it ran in when the run found only warnings. A
`--strict` run whose findings were all non-escalating printed
`SUCCESS — (tolerant mode) with N warning(s)`, which invited recording a strict pass
nobody claimed or re-running the gate believing the flag had been dropped. The
escalation is unchanged: `--strict` still turns a structural warning into an error.
