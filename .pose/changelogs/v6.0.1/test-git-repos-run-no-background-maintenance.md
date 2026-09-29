---
spec: test-git-repos-run-no-background-maintenance
category: fixed
breaking: false
---

The engine's test suite no longer fails at random with `TempDir RemoveAll cleanup: ... .git: directory not empty`. Git starts a detached `maintenance run --auto` after each commit, which kept writing into test repositories while they were being removed. The test binaries now run git with automatic maintenance disabled; the engine's own git invocations are unchanged.
