---
spec: pose-mcp-server-survives-self-update
category: fixed
breaking: false
refs:
---

A `pose serve-mcp` that was running when `pose update` replaced its binary can
run its CLI-backed tools again. It failed every one of them with
`fork/exec …/pose.old: no such file or directory` until it was restarted — a
message that reads as a missing binary, not as a server needing a restart.

The server found the CLI by calling `os.Executable()` on every call. `pose
update` renames the running binary to `.old` and removes it once the new one is
in place, and on Linux `os.Executable()` then names the removed file. It is the
defect v2.0.0 hit in the update handoff; the fix there stayed in the handoff,
and the one caller that outlives an update kept resolving.

The path is now resolved once, when the process starts. After an update it holds
the updated engine, which is what a restarted server would run as well.
`POSE_EXECUTABLE` still takes precedence.
