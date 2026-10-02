---
spec: pose-delivery-integrity-index-compaction
category: changed
breaking: false
---

Write the delivery-integrity index in schema 2: validation runs and result sets are stored once, repeated edges are written once and `changes` edges are derived from the change sets. On this repository the index shrinks from 10.9 MB to 4.6 MB for the same inputs, with the same provenance digest, claims, change sets and findings, and it now grows with deliveries rather than with deliveries times results. Gates and MCP tools read the same graph as before. An earlier engine refuses a schema 2 index as invalid until `pose index` rewrites it; this engine still reads schema 1.
