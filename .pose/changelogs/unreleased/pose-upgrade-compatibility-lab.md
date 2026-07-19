---
spec: pose-upgrade-compatibility-lab
category: added
breaking: false
refs:
---

`pose upgrade` now has full test coverage against a populated instance
(pt-BR locale content, a real spec, a knowledge note, a hand-edited
`AGENTS.md`): dry-run is proven byte-for-byte non-mutating, apply changes
only `schema-version` and preserves everything else, reapply is a strict
no-op, and an instance newer than the engine fails with explicit
remediation rather than a partial upgrade. `pose upgrade` also now refuses
to follow a symlinked managed directory instead of silently writing
through it. The release compatibility gate's real N-minus pairs get the
same populated-fixture depth once a supported upgrade is declared.
