---
spec: pose-validation-runtime-guardrails
category: added
breaking: false
refs:
---

`pose validate` now bounds every check with a timeout and an output ceiling,
killing the whole process group on breach and recording an explicit
`limit_state` — never conflated with a normal check failure. Checks marked
`isolation: "required"` are never run locally; `--emit-plan` exports a
Harness execution-plan envelope binding project, spec, matrix digest and an
approval slot instead.
