---
spec: pose-domain-rule-extension-migration
category: changed
breaking: false
refs: ISSUE#24
---

`backend-go.md` and `frontend-react.md` no longer ship as embedded core rules — they install exclusively as extensions (`pose-rule-backend-go`, `pose-rule-frontend-react`), the same pattern already proven by `pose-rule-kubernetes`. A repository without a Go backend or React frontend no longer receives an irrelevant rule by default. Already-installed instances keep whatever they already have; `pose doctor` now flags any machinery file that stopped receiving updates and names the extension to install to resume them.
