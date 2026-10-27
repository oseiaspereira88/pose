---
spec: pose-delegated-review-round-protocol
category: added
breaking: false
---

Make the rounds of a delegated review one process. From round 2 the generated brief (`review-brief/2`) carries the open findings of the previous round, each with an identity kept across rounds, and asks for a `pose-draft` block (`pose.review.draft/v1`) that accounts for every one of them and states its sweep; a draft that does not is recorded with `draft_problems`, reported `invalid` by `wait`, and never accepted as an approval. With `routing.rounds` `continue-thread` (the default) a round resumes the previous round's session, and a new one opens, with its reason, on an agent or vendor change, a broken chain, an unresumable session or `routing.round_threshold` (5). Runs record `round` and `thread`; `verify`, `ls` and `insights` show the round, the session lineage, the threshold warning and convergence per scope, split between continued and cold rounds.
