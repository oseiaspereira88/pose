---
spec: pose-attention-federated-parity
category: fixed
breaking: false
refs:
---

`pose state --attention` now reads obligations through the same governed store as MCP, so federated acceptance blockers that `pose_obligations` reported are no longer missing from the CLI for the same snapshot.
