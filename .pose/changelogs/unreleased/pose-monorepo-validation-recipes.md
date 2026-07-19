---
spec: pose-monorepo-validation-recipes
category: added
breaking: false
refs:
---

Published three executable monorepo recipes — JavaScript/npm workspace,
declared dependency graph (Bazel-style fine-grained modules) and a
mixed-language repository with a shared high-criticality module —
demonstrating changed-scope selection, severity and metadata together. Each
recipe is a docs-as-test: the same fixture and commands shown in the docs run
in CI, so the documentation cannot silently drift from the engine.
