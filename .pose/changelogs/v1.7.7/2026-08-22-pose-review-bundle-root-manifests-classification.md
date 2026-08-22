---
spec: pose-review-bundle-root-manifests-classification
category: fixed
breaking: false
---

Fixed review bundle path classification for root-level project manifests (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, etc.), toolchain configuration files, and root documentation files (`PROJECT.md`, `CLAUDE.md`, `LICENSE`, etc.). Supported single-module repositories rooted at `.` so their direct files and common source directories are properly classified without blocking bundle seal.
