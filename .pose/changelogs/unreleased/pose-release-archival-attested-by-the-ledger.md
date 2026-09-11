---
spec: pose-release-archival-attested-by-the-ledger
category: fixed
breaking: false
refs:
---

A spec that declared its changelog fragment now passes `pose artifact-check` after the release that archives it. `pose release prepare` used to rewrite the spec's claim into a rename no commit of the spec performed — failing every released spec with `action-mismatch`, and changing the sealed review subject of a spec closed before the cut. It no longer edits specs: the release manifest attests where the fragment went, and the check resolves the claim through it. Claims rewritten by earlier releases resolve the same way, with nothing to migrate.
