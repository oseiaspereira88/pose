---
spec: pose-skill-closeout-gate-parity
category: fixed
breaking: false
refs:
---

The pt-BR `pose-spec-closeout` skill now routes the closeout through the review
gate. It had gone from follow-up triage straight to hand-editing `status: done`,
never mentioning `pose review record`, `pose review-check` or `pose close` — so
an agent following the translated skill produced a spec that `pose check
--strict` then rejected with an error whose cause was not visible from its text.
The other ten translated skills were checked the same way and teach every
command their English counterparts do.
