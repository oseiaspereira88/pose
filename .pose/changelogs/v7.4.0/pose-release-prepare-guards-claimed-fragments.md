---
spec: pose-release-prepare-guards-claimed-fragments
category: fixed
breaking: false
refs:
---

`pose release prepare` no longer freezes a release over a changelog fragment that another spec still claims at its unreleased path, which left that claim pointing at a moved file and turned CI red after the v7.1.0 freeze. `release plan` names each such claim, `prepare --apply` refuses unless `--allow-moved-claims`, and `release check` reports it on a prepared tree.
