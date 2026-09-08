---
spec: pose-contract-adoption-registry
category: changed
breaking: false
refs:
---

Governance contracts whose rules judge completed work now come from one
registry, and a review policy records their adoption dates in
`contract_adoptions`. Declaring a contract there is enough for `pose update` to
stamp it and `pose doctor` to report it; neither command knows any contract by
name.

Existing policies are untouched. `component_aware_adopted_at`,
`review_bundles_adopted_at` and `evidence_vocabulary_reconciled_at` keep working
exactly as before, are read as that contract's adoption date, and are never
rewritten into the map.

Before this, each tightening of a rule cost a new policy field, a bespoke
exemption and — because `.pose/policy/` is not machinery — a release in which
every adopting instance saw its historical closeouts fail with no indication
that a date was what was missing.
