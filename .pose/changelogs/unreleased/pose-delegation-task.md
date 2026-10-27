---
spec: pose-delegation-task
category: added
breaking: false
---

Add `pose delegate task <prompt-file|->`: a general task handed to another agent in a real directory (`--cd`) with the access its caller chooses (`read`, `sandboxed`, `write`, `full`; `full` is never a default and warns), `--add-dir`, `--effort` and `--no-network` placed by the agent's template and refused when it cannot carry them. The final message goes to stdout and the agent's exit code is returned. The record is owner-only under the user's state directory, or under `.pose/delegate-runs/` with `--record`; no review reader reads a task and attestation refuses it. `ls`, `watch`, `wait`, `cancel`, `resume` (with a default continuation message or `--prompt`) and `handoff` work on tasks.
