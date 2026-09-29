---
spec: pose-cli-output-machine-channel
category: changed
breaking: false
---

`lint-spec`, `index`, `history-check`, `knowledge-check`, `skills-check`, `recurrence-check` and `state` accept `--json`, `--quiet` and `--color`, and every gate on the machine channel, `check` included, accepts `--json-out <path>` to write the same document to a file while keeping the human report. Their findings now go through the renderer: `history-check`, `knowledge-check` and `recurrence-check` findings that went to stderr as `[WARNING]`/`[ERROR]`/`[RECURRENT]` lines are now the gate's result on stdout. Verdict lines and `name=value` fields are unchanged.

**Deprecated:** `pose validate --json <path>`. Everywhere else `--json` prints to stdout; use `pose validate --json-out <path>`. The old spelling keeps working and prints a deprecation notice on stderr.

`pose report` no longer drops the first character of the first changed path (`README.md` was recorded as `EADME.md`), and the contributor hint goes to stderr in `check` and `artifact-check` too.
