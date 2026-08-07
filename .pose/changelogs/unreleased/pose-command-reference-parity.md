---
spec: pose-command-reference-parity
category: fixed
breaking: false
refs:
---

The POSE.md command reference now lists every command the CLI advertises — it
had drifted to roughly half of them, so shipped capabilities were invisible to
agents choosing what to run. `pose check --strict` fails from now on when a
manual omits an advertised command, `report-limitation` appears in `pose help`,
and the MCP reference gained worked `.mcp.json` configuration examples.
