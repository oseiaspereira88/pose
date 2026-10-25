# ADR: Recover supersession from one immutable predecessor intent

- Status: accepted
- Date: 2026-10-24
- Spec: spec:pose-action-request-recovery-and-preflight

## Context

B09 authorizes retirement but cannot converge two journals after an interrupted opening. Returning an existing successor skips retirement. Concurrent successor intents can otherwise race. Preview omits the domain checks and presentation repeats invalidated answers or hides declined/missing-subject conditions.

## Decision

Persist one immutable schema-1 intent under `.pose/actions/supersessions/<predecessor>.json`, containing the predecessor revision/digest, prepared successor identity and a digest of the semantic retirement payload. Hold the existing nonblocking predecessor journal lock from admission through reservation, exclusive successor creation and retirement append. Never lock two journals; readers take no locks. Make one attempt per command, without automatic retries. A matching retry resumes the remaining phase; a competing intention or changed predecessor requires explicit repair, never overwrite. Bound intent files to 64 KiB and validate regular paths, schema, digest and lineage. Preserve malformed/torn journals for the existing supported repair workflow. Read phases from durable records: reserved, successor-opened, complete or repair-required; these are operation progress, not new action lifecycle states. Both records remain visible and restrictive while incomplete. A completed replay acknowledges recorded history; an incomplete replay revalidates current B09 authority/proof before any write.

Share resolution preparation between preview and apply. Missing required proof is pending in a read-only preview, with acceptance explicitly not certified; supplied bad proof is refused. Recheck structural preconditions, policy, proofs and state under the protected append, including answers. Exact semantic idempotency excludes emitter bytes so a renewed proof can acknowledge the same operation, but conflicting payloads cannot. Allow recipient-authorized waiver of a declined, materially current condition; invalidated/missing content requires replacement/repair.

Expose one typed next-step projection in request views, presentation, obligations/Attention and CLI/MCP. Do not offer an old-digest answer for invalidation, auto-repeat a declined question or re-ask a satisfied decision. Refusal is a typed command result, with `audit_persisted: false`; do not append refusal noise to the question journal. This prospectively reconciles R2 of `pose-action-request-resolution`, whose phrase “the attempt is recorded as refused” did not result in durable refused events in the implementation. It supersedes the expectation of automatic refused-event auditing; the historical event vocabulary remains readable, without fabricated retrospective entries.

## Alternatives

- Scan all requests for competing successors: rejected because recovery should be bounded by one predecessor, not repository size.
- Treat sequential appends as a transaction: rejected because creation/Sync/crash can interrupt the sequence.
- New general transaction engine or automatic journal truncation: rejected; B13/B14 own general recovery/publication.
- Skip proof on partial retry: rejected because a prior preview/opening is not current retirement authority.

## Consequences

The public preview/refusal and competing-intent semantics are breaking and need release compatibility review before publication. No policy or issuer is adopted. Upgrade all writers sharing these journals before adoption; older writers do not enforce the new intent reservation. Filesystem replacement by arbitrary external writers remains outside authenticated historical authorization. A corrupt intent or partial journal yields repair-required, not automatic repair. Revisit this decision if transaction storage or the action audit policy changes.
