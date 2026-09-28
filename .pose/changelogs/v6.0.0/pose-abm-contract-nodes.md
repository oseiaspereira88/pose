---
spec: pose-abm-contract-nodes
category: added
breaking: false
refs:
---

`pose amend <slug> --nodes [--json]` and the `pose_spec_amendments` MCP tool expose a versioned contract-node projection of a spec: requirements, assumptions and decisions with namespace, content hash, state and relations (schema `v1/contract-nodes.schema.json`). Decisions may declare `Status: active|withdrawn`. Amendment events gain schema 2, which records R/A/D nodes with their state before and after, a `transition` change and `assurance: declared`; schema-1 logs keep their meaning. Adopting `contract_nodes_version: 1` in the review policy extends the amendment gate to assumptions and decisions, refuses an editorial acknowledgement of a changed state or relation set, and rejects an active decision resting on an invalidated or withdrawn assumption. Without the capability nothing changes, and a schema-2 event is refused instead of read as requirement history. No instance adopts the capability in this release.
