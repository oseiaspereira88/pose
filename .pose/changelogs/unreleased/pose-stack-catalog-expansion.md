---
spec: pose-stack-catalog-expansion
category: added
breaking: false
refs:
---

POSE now detects and validates Python (poetry, pipenv, pip, setuptools,
PEP 517) and .NET repositories alongside Node.js, Go, Rust and Java. The new
`pose stacks` command reports every matched manager per directory —
prerequisite availability, confidence and which marker wins when several
managers' files coexist — entirely offline, without executing any project
file. Detection feeds the existing wizard and validation matrix unchanged.
