---
spec: pose-delegation-task-detach-and-state
category: added
breaking: false
---

Add `pose delegate task --detach`: a task starts as a detached run and returns its attempt, its log and the place of its record; `wait`, `ls`, `watch` and `cancel` resolve it, and a route that must first wait for a usage-limit reset is not detached. A prompt piped to `pose delegate task -` or `pose delegate resume --prompt -` now reaches the agent — the parser took a lone `-` for an unknown flag and `--prompt -` for a missing value — and a detached run receives it in a private file instead of reading the caller's stdin. The task state directory takes `XDG_STATE_HOME` only as an absolute path, as the XDG Base Directory specification requires (a relative value, which placed records relative to the working directory, now falls back to `~/.local/state`, and records written there are no longer listed), and refuses to record or read under a directory another user could rewrite: the resolved base and its ancestors must belong to the user or root and be writable by no group or other user unless sticky, and the `pose/delegate/<project>` directories must be the user's real, non-writable directories.
