---
spec: pose-action-request-resolution
category: added
breaking: false
refs:
---

`pose action resolve|cancel|waive|invalidate` records answers that count only when they should: the actor must hold the recipient role in `.pose/policy/actions.json`; the write names the request digest it answers (a changed request or subject makes an earlier answer insufficient) and the revision it read (a second answer against the same revision is refused); an idempotency key makes retries no-ops and refuses the same key with other content. Under verified assurance a resolution needs an Ed25519 claim from a pinned issuer bound to project, audience, digest, principal and answer, so an agent writing `human:<name>` is not a verified confirmation. A declined approval is recorded and authorizes nothing; the requester may cancel its own question, and a waiver needs the authority and a reason.
