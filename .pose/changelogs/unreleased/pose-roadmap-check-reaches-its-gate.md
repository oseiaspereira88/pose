---
spec: pose-roadmap-check-reaches-its-gate
category: fixed
breaking: false
refs:
---

`pose roadmap-check` reported zero cut criteria and exited 0 — in both
`--strict` and `--tolerant` — on a repository whose roadmap declared several.
The delivery graph builder returned before it loaded a single roadmap when
`.pose/indexes/validation-matrix.json` or `.pose/specs` was absent.

A gate that cannot evaluate its criteria was reading as a gate that passed
them. Neither absence justifies the skip: without delivery profiles there are no
targets, so a criterion naming one is unresolved — which the criteria loop
already reports as `unknown delivery ref` — and a `check:` or `manual-review:`
ref needs no profile index at all.

A repository that has adopted delivery integrity is unaffected: it never took
the early return, and `surface-check` here reports the same edges and findings
before and after. What changes is the answer given to a repository with roadmaps
and no profile index, where the previous answer was wrong.

Found while writing a coverage fixture rather than by a report: the fixture had
no profile index, and both modes returned 0 on a criterion naming a delivery
target that does not exist.
