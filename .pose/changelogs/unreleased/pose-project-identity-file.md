---
spec: pose-project-identity-file
category: added
breaking: false
refs:
---

A project declares its id once in `.pose/project.json`. `pose install` writes it and `pose update` seeds it from the id an older instance declared in `.mcp.json` or `AGENTS.md`, so the CLI, the MCP server and every checkout resolve the same project under any directory name. An environment binding that disagrees with the file is refused as `conflicting-project-binding`; `pose doctor` reports the declared identity under `project.identity`, and the directory-name limitation now names the file as the remedy.
