---
spec: pose-release-notes-name-contract-adoption
category: added
breaking: false
refs:
---

A release that introduces a governance contract now says so, in a Compatibility
section at the top of its own notes: which contracts, what each requires, and
what adopting one costs a reader that does not have it.

The cost is stated per contract, because it differs. An engine older than the
release never applies the contract — it does not know the id — and judges
reviews by the rules it has. On top of that, a contract carried by a top-level
policy key is refused outright by engines from before the strict decoder was
dropped in 2.0.2: they stop reading the repository until they are updated, so
every tool has to move together. A contract recorded as an id inside
`contract_adoptions` costs none of that, because the map is a key those engines
already model.

v2.0.0 introduced one of the first kind and its notes said none of it.

The text comes from the contract registry, which is already the one place a
contract is declared — adding one there is what makes `pose update` stamp it and
`pose doctor` report it, and now also what the notes say. A contract added
without recording which release introduced it fails a test, so the next one is
not silent.

Releases already prepared keep their frozen notes and their digests: nothing
re-renders them.
