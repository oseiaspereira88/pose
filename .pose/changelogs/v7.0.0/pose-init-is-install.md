---
spec: pose-init-is-install
category: fixed
breaking: false
refs:
---

`pose init` in a repository without POSE now runs the full installation — the same instance `pose install .` produces — instead of creating empty directories that `pose check` rejects and `pose new-spec` cannot use. On an installed instance it writes nothing and names `pose update`. The installer's `--locale`, `--project-name`, `--project-id`, `--skip-mcp` and `--allow-non-git` flags pass through, and `--wizard` runs after the installation.
