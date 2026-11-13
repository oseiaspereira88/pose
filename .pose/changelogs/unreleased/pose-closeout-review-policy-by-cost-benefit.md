---
spec: pose-closeout-review-policy-by-cost-benefit
category: added
breaking: false
---

Add the closeout review policy (review policy schema 5): `milestone_review_required` and `roadmap_review_required` take `never`, `when-justified` or `always`, adopted with `pose adopt closeout-review-policy` (new instances default to `never`/`never`; existing policies keep the legacy rule until they adopt). One evaluator, `closeout-review@1`, answers for review plans, verification, closeout state, attention, doctor and MCP: `when-justified` requires a milestone review above two distinct canonical specs and a roadmap review above one spec with a resolved ADR association (`adrs:` on the roadmap, `roadmaps:` on the ADR, or a local Markdown link). Under adoption `pose close` of a milestone or roadmap stays explicit and publishes a receipt under `.pose/review-closeouts/` recording `review: not-required` or `required`, the key, value, evaluator and reason; unknown members or associations never exempt, a recorded rejection stands, `always` overrides singleton inheritance and `unreviewed_milestones`, and scopes closed before adoption keep their sealed reading. Milestone members written as `spec:<slug>` now resolve as their slug in closeout state.
