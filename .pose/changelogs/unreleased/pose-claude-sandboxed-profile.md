---
spec: pose-claude-sandboxed-profile
category: added
breaking: false
---
The Claude agent of the delegation policy offers a `sandboxed` access profile, as Codex does, using the Claude Code operating-system sandbox. The profile refuses to start, naming which of bubblewrap and socat is missing and how to install it, rather than letting Claude Code run its commands unsandboxed.
