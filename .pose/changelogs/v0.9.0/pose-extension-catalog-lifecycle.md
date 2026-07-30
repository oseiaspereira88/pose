---
spec: pose-extension-catalog-lifecycle
category: added
breaking: false
refs:
---

Teams can now install, list, remove and verify third-party skills,
workflows, rules and import adapters without forking POSE:
`pose extension install/list/remove/verify` reads a versioned manifest
(contents, compatibility, permissions, conflicts, provenance), requires a
Sigstore signature by default, and applies changes transactionally —
dry-runnable, consent-gated and fully rolled back on any failure.
User-modified files are never silently overwritten or removed. Extensions
are data only; nothing is ever executed on install.
