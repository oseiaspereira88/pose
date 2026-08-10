---
spec: pose-scaffold-allowlist
category: changed
breaking: false
refs:
---

The embedded scaffold is now an allowlist: only what `pose install` and
`pose upgrade` actually read is carried in the binary. It included by default,
which is why a published product contract nearly shipped to every instance — and
why the binary carried this project's own specs, ADRs, reviews, changelogs and
release manifests, megabytes it never opened. The embedded tree drops from 514
files to 104 and the binary from 27.5 MB to 26.2 MB, with the installed instance
byte-identical to before.
