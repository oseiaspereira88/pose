---
spec: pose-mcp-catalog-conformance
category: changed
breaking: false
refs:
---

The MCP tool catalog is now a tested public contract: a reviewed golden
fixture freezes every tool name and input schema, each tool declares a risk
class (read, gate or external-side-effect), and the optional Conductor
reporter tools document their activation conditions. Documentation and
registry metadata are checked against the same catalog on every build.
