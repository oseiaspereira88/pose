---
spec: pose-spec-amendment-history
category: added
breaking: false
refs:
---

Specs gain an append-only amendment history: `pose amend` records material
requirement changes with affected IDs, rationale, author/reviewer and
timestamp, and the closeout gate rejects any requirement rewritten after its
evidence without an acknowledging event. Editorial rewording is a one-line
acknowledgment; the MCP tool `pose_spec_amendments` projects the history.
