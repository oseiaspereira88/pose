---
spec: pose-doctor-fixtures-exercise-production-path
category: changed
breaking: false
refs:
---

Three `pose doctor` branches that no test executed are now covered: the tool
half of `review.evidence-vocabulary`, which inspects a profile's criteria and
its tools and only ever ran the criteria half; the `ok` answer of
`policy.artifact-roots`, which had only ever been proven to warn; and
`mcp.config` for a configuration that exists and points somewhere other than the
native binary.

They were found by measuring which statements the suite executes rather than by
re-reading the tests, which is how the same class of gap survived twice.
