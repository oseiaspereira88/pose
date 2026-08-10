---
spec: pose-locale-coverage-contract
category: fixed
breaking: false
refs:
---

Translation coverage is now a property of the locale tree rather than a list of
documents someone remembered. Every file under `locales/` is discovered by
walking it, must have an English source, and must have a declared comparison —
so a new translated document is guarded by default instead of guarded after
someone reads it and notices the drift.

That default found nine real disparities. The pt-BR review and feature workflows
did not teach the closeout gates they describe (`pose review record`,
`pose review-check`, `pose closeout-check`, `pose close`), the recurrence
workflow did not name `pose recurrence-effect`, and the delivery-evidence rule
omitted `pose roadmap-check`. Three findings ran the other way and corrected the
English, which now names the MCP tools and the handoff command the translation
already did.

One finding was not drift but deletion: `locales/pt-BR/.pose/rules/kubernetes.md`
translated a rule the engine no longer owns. It left the tree when the Kubernetes
rule became a signed extension, and the translation shipped to every instance for
eight releases after. It is withdrawn.

The command extractor also matched line by line, so a code span wrapped across
two lines read as untaught — reflowing a paragraph disarmed the check silently.
Fixing that immediately surfaced an omission the false negative had been hiding.
