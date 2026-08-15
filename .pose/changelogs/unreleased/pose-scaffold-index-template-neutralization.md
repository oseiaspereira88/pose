---
spec: pose-scaffold-index-template-neutralization
category: fixed
breaking: false
refs: ISSUE#22
---

A fresh `pose init`/`pose install` no longer seeds `.pose/indexes/module-metadata.json` and `.pose/indexes/validation-matrix.json` with POSE's own development-repository module entries (`pose-mcp`, `mcp-enforce`, `docs-site`, `@pose-maintainers`). The generic, reusable check catalog (`stacks`, `deliveryProfiles`) still ships as before — only the pose-mcp-specific module graph and overrides are neutralized.
