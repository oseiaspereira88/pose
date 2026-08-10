---
spec: pose-skill-index-parity
category: fixed
breaking: false
refs:
---

The pt-BR skill index lists every skill again. `.agents/skills/README.md` is the
routing table an agent reads to discover what exists, and the translation was
missing `pose-surface-closeout` and `pose-release-closeout` entirely — two
skills a pt-BR reader had no way to find. The command-parity gate compared
`SKILL.md` files and walked straight past the index that points at them; it now
covers both.
