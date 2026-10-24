# ADR: Bind action lifecycle proof to operation and predecessor

- Status: accepted
- Date: 2026-10-24
- Spec: spec:pose-action-waiver-and-supersession-authority

## Context

Answer proofs do not authorize removing a condition. Waivers currently accept labels; opening a replacement can supersede an unrelated request. The journal reader verifies answers only.

## Decision

Introduce an additive lifecycle binding version 1 on new cancellation, waiver and supersession events. A schema-2 issuer claim or a `pose-action-lifecycle` SSHSIG signs operation, request/project/digest, predecessor revision, reason, idempotency key and successor identity/digest. Keep schema-1 answer bytes and verification unchanged. Both supplied emitters must verify. Recheck authority and proof under the predecessor journal lock before writing.

| Operation | Authority | Proof | Effect |
| --- | --- | --- | --- |
| answer | recipient/role | existing policy/channel | admitted answer only |
| waive | recipient/role | lifecycle proof under verified or MCP | dispense this condition |
| cancel | requester only | lifecycle proof under verified or MCP | withdraw this question only |
| supersede | predecessor requester or recipient/role | lifecycle proof under verified or MCP | replace this request with bound successor |
| material invalidation | engine subject resolver/transfer | observed content or governed transfer provenance | unmet condition, never waiver |

Declared local CLI/domain operations remain declared under policy off. MCP requires proof for actor lifecycle writes, regardless of labels. New actor invalidation is refused; existing engine subject comparison remains the observable provenance. Historical events lacking lifecycle bindings are disclosed as legacy, never upgraded by their stored assurance; unsigned legacy exemptions do not satisfy a verified policy. Existing issuer pins/human grants and historical time precision remain authoritative. Registered SSH keys authorize new writes; historical signatures preserve journal-only verification after rotation.

Supersession authorizes predecessor retirement before creating the successor, then revalidates at the protected append. Cross-journal convergence/recovery remains B10: an explicitly reported partial opening cannot withdraw the old condition.

## Alternatives

- Reuse an answer signature: rejected because it permits cross-operation replay.
- Require signed mechanical observations: rejected because content comparison has no new human decision.
- Treat every target as authorization: rejected because relevance does not grant retirement authority.
- Build a new issuer or multi-journal transaction: rejected as outside B09; reuse trust and leave convergence to B10.

## Consequences

The stricter lifecycle semantics require a minor or breaking release assessment, never a patch claim by default. No instance policy, issuer, release or consumer pin is adopted. Arbitrary filesystem writers remain outside the proof of engine observation; persisted proof does not prove historical role authorization. Restoration uses Git without rewriting recorded journals; rejected attempts must preserve journal bytes.
