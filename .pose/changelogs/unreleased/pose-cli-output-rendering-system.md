---
spec: pose-cli-output-rendering-system
category: changed
breaking: false
refs:
---

The CLI gains one rendering layer: every command prints through it, with one
severity vocabulary, one channel rule (results on stdout, progress and failures
on stderr), symbols with an ASCII fallback, and colour only on a terminal.
Long-running commands report each step with its outcome and duration — a live
status line on a terminal, one line per event anywhere else — and `pose validate`
shows a failing check's output instead of streaming every check's. `--json`
prints to stdout on every command with a result, `--json-out` writes files, and
`--quiet` prints the verdict alone.
