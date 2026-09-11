---
spec: pose-docs-and-manuals-match-5-0-2
category: changed
breaking: false
refs:
---

The docs site, POSE.md, the spec template, the rules, the workflows and the
skills describe 5.0.2. The site had not been revised since before v1.8.0: it now
documents what a sealed review bundle fixes and what an attestation has to show,
the closed evidence vocabulary, every `pose doctor` diagnostic, `pose install`,
`pose public-claims`, release Compatibility sections, and the MCP server's
lifecycle, and gains an architecture mechanism for component-aware review.

Checking each claim against the code also corrected errors: `pose update` has no
`--schema-only` (it has `--no-self`) and merges POSE.md/AGENTS.md on every run;
`pose new-spec` writes a dated flat file by default; `pose dora-metrics` takes
`--window-days`; and the Definition of Ready is opt-in through
`.pose/policy/dor.json`. The spec template gains `task_type`, which the
Definition of Ready reads.
