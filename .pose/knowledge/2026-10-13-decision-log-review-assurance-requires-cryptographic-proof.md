---
type: decision-log
slug: review-assurance-requires-cryptographic-proof
sensitivity: public-internal
created_at: 2026-10-13
owner: "@pose-maintainers"
last_reviewed_at: 2026-10-13
expires_at: 2026-11-13
source_refs:
  spec: "pose-review-disclosure-requires-proof"
  external_sources: []
---

# Decision log: Authenticate review proof before disclosing assurance

## Context

B01/T2-03 identified different trust paths: the gate checked the signed envelope, while disclosure called an authority checker that only compared claim fields. A claim with coherent fields and a damaged signature could therefore be disclosed as verified although approval was refused. The synthetic regression starts from a trusted valid fixture, damages only the envelope signature, and expects both the verdict and disclosure to refuse verified proof. The old branch has no cryptographic check on the disclosure path; local test execution is unavailable because the Go compiler is truncated.

## Decision

Verify the stored signature and exact supplied bundle binding inside the shared authority checker. Let the declared path retain its existing signature gate. Keep criteria and decision validation outside authority verification, so a valid proof does not become invalid identity merely because approval fails.

## Consequences

Direct disclosure, verify, review-check, closeout and attribution consume the same authority proof check. Invalid proof never yields a verified claim that could authenticate human confirmation. No records, policy, schema or capability adoption are rewritten. Current issuer pins/grants and current expiry semantics remain; historical trust and reuse provenance require the separate B04 delta.

## References

- spec:pose-review-disclosure-requires-proof
- knowledge:action-answer-proof-and-historical-time
- .pose/reports/2026-10-11-pose-front-02-trust-chain.md
