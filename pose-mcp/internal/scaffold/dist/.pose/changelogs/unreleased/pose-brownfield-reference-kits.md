---
spec: pose-brownfield-reference-kits
category: added
breaking: false
refs:
---

Three real, executable adoption kits now live under
`examples/brownfield-kits/`: direct adoption, GitHub Spec Kit import and
OpenSpec import/reconciliation, each a small checked-in brownfield fixture
with a staged visibility-to-blocking-gate guide. Every kit is exercised
end to end by the test suite (preservation of pre-existing content,
surfaced curation warnings, DoR readiness, git-native rollback safety),
so the guides cannot silently drift from what the CLI actually does.
