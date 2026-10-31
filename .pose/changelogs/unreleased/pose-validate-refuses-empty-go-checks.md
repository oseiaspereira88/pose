---
spec: pose-validate-refuses-empty-go-checks
category: changed
breaking: false
---
`pose validate` fails a `go test` check that selects tests with `-run` or `-skip` when Go's result lines report every package as having run nothing, with `limit_state: no-tests`, instead of passing on Go's exit 0. A project with a check that selects nothing starts failing it.
