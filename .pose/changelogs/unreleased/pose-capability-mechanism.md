---
spec: pose-capability-mechanism
category: added
breaking: false
refs:
---

Adds the capability assessment as a POSE-native mechanism: a structured, versioned artifact at `.pose/capabilities/assessment.md` (flat frontmatter + per-mechanism flat bullets) with typed, mechanically verified evidence references; an append-only `history.jsonl` snapshot log; and the `pose assess` command family (`init` scaffolds the 16 default mechanisms, bare `assess` validates structure/evidence/stable-ids/staleness, `snapshot` appends, `diff` compares). `pose check --strict` runs the same validation when the artifact exists (opt-in by presence). The repository's own assessment (2026-07-19) is migrated as dogfooding.
