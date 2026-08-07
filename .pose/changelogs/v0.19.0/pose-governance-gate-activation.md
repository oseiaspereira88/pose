---
spec: pose-governance-gate-activation
category: changed
breaking: true
refs:
---

Technical-debt marker ids now derive from the marker itself — its file, keyword
and line text — instead of its position in the scan. Adding a marker earlier in
a file no longer renumbers every later one, so a spec or review citing an id
keeps pointing at the same debt. Existing ids change shape once, exactly as
`gap_id` did in 0.18.0.

Three governance gates that could never fail now can. A `done` spec with
requirements and no `### Requirement trace` section is an error rather than a
warning — every shipped spec was given a trace first. The quarterly governance
audit runs `followups --all --fail-overdue`, so an overdue follow-up fails it.
Deterministic validation results are published to GitHub code scanning as SARIF,
next to the CodeQL findings.

The feature and bugfix workflows and skills now name the exact `knowledge:<slug>`
citation form that `pose knowledge-usage` counts; prose naming a file was never
counted, so consulted knowledge could look unused and expire on TTL.
