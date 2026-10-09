---
spec: pose-trace-test-refs-resolve
category: changed
breaking: false
refs:
---

`lint-spec` and `pose close` now check that every `test:` ref in a requirement trace names a test that exists — a Go test or subtest, a JavaScript/TypeScript test title, a Python or Rust test, a tracked test file, or a `go test -run` pattern — in the repository, its submodules or a nested module repository the validation matrix registers. An unresolved ref fails an open spec and is only reported on a closed one, so history is not rewritten.
