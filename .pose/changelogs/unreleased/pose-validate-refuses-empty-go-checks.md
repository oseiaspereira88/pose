---
spec: pose-validate-refuses-empty-go-checks
category: changed
breaking: false
---
`pose validate` fails a `go test` check with a `-run` pattern that ran no test in any of its packages, with `limit_state: no-tests`, instead of passing on Go's exit 0. A project with a check that selects nothing starts failing it.
