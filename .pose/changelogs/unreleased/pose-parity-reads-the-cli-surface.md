---
spec: pose-parity-reads-the-cli-surface
category: changed
breaking: false
refs:
---

The locale parity checks learn what a pose command is from the CLI's own
dispatch, read from its source.

The skill check kept a hand-written list of eight commands that take a
subcommand; the CLI has thirteen. `contribute` was missing, so a translation
teaching `contribute submit` where the English skill teaches `contribute stage`
passed — and `roadmap`, which is not a command, was on the list. The manual
check recognised a command only when a manual listed it as an entry, so seven
commands AGENTS.md mentions only in prose had no signal.

Both failed quiet. No shipped skill or manual had drifted behind them.
