---
spec: pose-abm-causality-attestation
category: added
breaking: false
refs:
---

Adopting `causality_closeout_version: 1` in the review policy stamps a `causality-closeout` contract on new review bundles, sealing the plan band and the contract-node digest of each spec in scope so a material R/A/D change stales the review. Under it, a structural mapping to an invalidated or withdrawn assumption, or a withdrawn decision, is refused; a `mapped` answer needs its own rationale and one reason pasted across three or more facts on the same basis is refused; `accepted-risk` needs an owner and review date (`--mapping …|accepted-risk|<why>|@owner|YYYY-MM-DD`) and cannot waive a public or governance contract; `not-applicable` is refused under unknown coverage; and an elevated or critical band with material facts and no answering criterion is refused, with no mandatory human. Bundles sealed without the stamp keep their verdict, and no instance adopts the capability in this release.
