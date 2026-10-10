---
spec: pose-review-subject-classifies-engine-records
category: fixed
breaking: false
refs:
---

`pose review bundle --seal` no longer refuses a scope that carries the signed legacy ledger `pose adopt signed-attestations` writes under `.pose/review-ledgers/`; the ledger is reviewed as governance. Delegated review runs, DORA events, investigation notes and `.pose/schema-version` are classified too, and a test keeps every directory the engine writes under `.pose/` classified.
