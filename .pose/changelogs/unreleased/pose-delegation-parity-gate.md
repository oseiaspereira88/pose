---
spec: pose-delegation-parity-gate
category: removed
breaking: true
---

Remove `pose review dispatch`, `pose review resume` and `pose review status`, the aliases `pose-delegation-surface` deprecated: each now exits 2 and names its replacement — `pose delegate review|adjudicate|smoke <scope|bundle-id> --agent <name>` for `dispatch --via <name>` (`--kind adjudication` is `pose delegate adjudicate`, a run without `--apply` is `--dry-run`), `pose delegate resume <run>` for `resume --apply`, and `pose delegate ls` for `status`. This release is the first to carry their removal and no release printed their deprecation, so a script or skill that calls them must move to `pose delegate`. `pose delegate review|adjudicate|smoke` now discloses the containment a run runs under (`delegate.safety`, and `safety` in `--json`), as `dispatch` did. Add the launcher parity gate: `.pose/parity/delegation-launchers.json` maps every capability the delegation ADR lists for the personal launchers to its `pose delegate` command and covering tests, and the `launcher-parity` check fails while a row is uncovered or an active document, skill or help text names a replaced command or launcher; the operating manual describes how an operator retires a personal launcher. The `pose delegate task`, `resume` and `handoff` help now documents every flag they accept.
