---
spec: pose-release-notes-name-contract-adoption
category: added
breaking: false
refs:
---

A release that introduces a governance contract now says so, in a Compatibility
section at the top of its own notes: which contracts, what each requires, and
that engines older than this release can no longer read the repository's review
policy once an instance adopts one.

That is a property of adopting a contract, not of how the adoption date is
encoded. Dropping the strict policy decoder stopped a *new field* from breaking
an older binary; it does nothing for the contract itself, because the key names
something the older engine does not have. Every tool reading the repository has
to move together, and v2.0.0's notes said none of it.

The text comes from the contract registry, which is already the one place a
contract is declared — adding one there is what makes `pose update` stamp it and
`pose doctor` report it, and now also what the notes say. A contract added
without recording which release introduced it fails a test, so the next one is
not silent.

Releases already prepared keep their frozen notes and their digests: nothing
re-renders them.
