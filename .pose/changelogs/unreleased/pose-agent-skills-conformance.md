---
spec: pose-agent-skills-conformance
category: added
breaking: false
refs:
---

Every shipped skill now declares POSE compatibility metadata
(`pose_schema_range`, `clients`, `capabilities`) and is validated by a new
`pose skills-check` gate — required fields, layout, confined link
resolution, an offline unsafe-instruction/secret-shaped-content scan, and a
`claude-code` symlink cross-check. Wired into CI and exposed read-only over
MCP as `pose_skills_check`. The gate immediately found and fixed a real,
pre-existing broken link.
