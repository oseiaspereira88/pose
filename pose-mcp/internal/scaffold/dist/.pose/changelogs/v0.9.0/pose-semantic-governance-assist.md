---
spec: pose-semantic-governance-assist
category: added
breaking: false
refs:
---

New `pose semantic-suggest` gives advisory, explainable suggestions of
related follow-ups, recurrence patterns and knowledge for a spec — every
suggestion cites its artifact with a score, a shared-terms rationale and
a provider ("lexical", the only approved provider, deterministic and
offline). Sensitivity-restricted knowledge is filtered before retrieval,
never suggested. `pose suggest-feedback` records an accept/reject
decision without the candidate's content, feeding future evaluation.
Suggestions never gate a check or mutate a spec.
