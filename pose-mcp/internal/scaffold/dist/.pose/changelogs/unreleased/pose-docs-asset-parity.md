---
spec: pose-docs-asset-parity
category: added
breaking: false
refs:
---

Release verification now checks that every download URL the documentation hands
a user names an asset the release actually publishes. The Homebrew formula was
offered as a copyable command for four releases while never being uploaded, and
nothing caught it because both sides were individually correct — the docs
described the intended distribution, the release published what its workflow
was told to. Nobody compared them.
