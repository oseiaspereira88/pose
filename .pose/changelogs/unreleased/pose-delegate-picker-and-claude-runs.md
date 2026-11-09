---
spec: pose-delegate-picker-and-claude-runs
category: fixed
breaking: false
---
The picker of `pose delegate watch` no longer fails in fish: its preview command is one `sh -c` word that every login shell parses, and the list now shows a state glyph, the agent and the age in aligned columns under a header, with the label last.
