---
spec: pose-lint-spec-flat-layout-followup-resolution
category: fixed
breaking: false
---

Fixed `pose lint-spec` followup disposition and sibling status resolution to properly discover specs formatted in the flat dated layout (`.pose/specs/<date>-<slug>.md`). Corrected `specsDir` navigation in `lintSpecFile`, expanded `cmdLintSpec --all` to list all specs across layouts, and supported flat specs in release claim repointing.
