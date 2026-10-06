---
spec: pose-legacy-contract-cutoffs
category: fixed
breaking: false
refs:
---

The review policy now reads `explicit_judgment_adopted_at` and `structural_causality_adopted_at`, so the cutoff an instance declared for the two 6.0.0 contracts takes effect for unstamped history; `contract_adoptions` still wins, sealed bundles keep their stamped contracts, an unparsable date is refused with its key named, and `pose doctor` shows where every contract adoption comes from.
