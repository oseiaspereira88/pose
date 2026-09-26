---
spec: pose-federated-spec-dependency-acceptance
category: fixed
breaking: false
refs:
---

A spec that depends on another project's artifact through `xref:` now seals the federated manifest of those dependencies into its review bundle, and `closeout-check` reports their blockers. Before, only roadmap and milestone scopes consulted federation, so revoking consumer trust left such a spec closable. Revocation now blocks the spec and stales its approved review; restoring the same trust restores it. Missing project roots block closeout with `project-roots-unavailable` instead of failing the command. Specs with no external dependency keep a byte-identical bundle.
