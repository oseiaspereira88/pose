---
spec: pose-cross-repo-portfolio
category: added
breaking: false
refs:
---

New `pose portfolio-projection` reconciles dependencies, readiness,
ownership and criticality across repositories registered via the same
allowlist the MCP server already uses (`HARNE8_PROJECTS_DIR` /
`POSE_PROJECT_ROOTS`) — repositories remain authoritative, the projection
is a read-only reconciled view. Specs can declare
`depends_on: xref:<project_id>/<spec-slug>`, additive to the existing
local reference forms. Every blocked, stale or unauthorized/unknown
cross-reference is explained explicitly, never fabricated as capacity or
silently dropped; disappeared artifacts are tombstoned across runs.
