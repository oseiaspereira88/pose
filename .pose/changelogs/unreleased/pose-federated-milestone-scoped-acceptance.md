---
spec: pose-federated-milestone-scoped-acceptance
category: fixed
breaking: false
refs:
---

Review and closeout of a milestone now consult federated acceptance over that milestone's own edges: the roadmap's `depends_on` and `consumes`, plus the milestone's `after`, `specs` and `consumes`. Before, a milestone used the acceptance of its whole roadmap, so an open spec in a later milestone blocked the closeout of every earlier one. Later milestones and the roadmap still report their own open members, and revoked trust still blocks. Milestone bundles sealed with the roadmap-wide manifest stay fresh while that sealed snapshot is unchanged; new seals carry the milestone as coordinator.
