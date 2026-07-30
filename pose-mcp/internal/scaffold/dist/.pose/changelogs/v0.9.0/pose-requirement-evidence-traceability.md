---
spec: pose-requirement-evidence-traceability
category: added
breaking: false
refs:
---

Specs can now trace every requirement to its evidence: a `Requirement trace`
section maps each stable R-ID to `[satisfied]` (with `check:`, `test:`,
`report:` and `commit:` refs), `[waived: reason]` or `[withdrawn: reason]`.
`pose lint-spec --strict` enforces full coverage at closeout, rejects orphaned
entries, and the new MCP tool `pose_requirement_trace` exposes the
bidirectional requirement↔evidence projection.
