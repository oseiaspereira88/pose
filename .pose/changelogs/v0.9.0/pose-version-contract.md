---
spec: pose-version-contract
category: fixed
breaking: false
refs:
---

Every public version surface — `pose version`, MCP `serverInfo.version` and
the registry metadata in `server.json` — now derives from one authoritative
source stamped at release time. Development builds always identify themselves
with a `-dev` suffix, and a contract test fails the build on any divergence.
