---
spec: artifact-claims-stop-at-the-section-end
category: fixed
breaking: false
---

The `### Artifacts` section now ends at a heading of any level. It ended only at
another `### `, so a spec whose artifact list was followed by `## 4. Tasks` — the
shape `pose new-spec` scaffolds — had its task checklist parsed as artifact claims,
and `artifact-check` and `roadmap-check` refused the whole run with
`malformed artifact claim "- [ ] ..."`. Claim sets for the 188 specs in this
repository that declare artifacts are unchanged.
