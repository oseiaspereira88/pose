# Adversarial corpus

Spec: `pose-mechanization-adversarial-corpus`. Test: `adversarial_corpus_test.go`.

Every gate POSE adds against mechanical compliance can itself be satisfied
mechanically. Each row is the cheapest formal satisfaction of one gate. An
`enforced` case fails the build when the shortcut works again; a `known-gap`
case fails the build when the gap closes, so it cannot close or reopen
silently; a `planned` case belongs to a spec not yet delivered and gains code
before that spec closes.

This corpus belongs to developing POSE's governance. It adds no checklist to an
adopting project's features.

| Case | Status | Shortcut it refuses | Owner |
|---|---|---|---|
| `human-label-without-confirmation` | enforced | An agent writes and applies a review under `human:<name>`; the record reads as a human review | pose-review-attribution-roles |
| `declared-independent-prefix` | enforced | `agent:independent-*` under declared assurance reads as authenticated separation | pose-review-assurance-disclosure |
| `confirmation-reused-for-edited-content` | enforced | A confirmation recorded for one set of conclusions is kept after they change | pose-review-attribution-roles |
| `unknown-recorded-as-satisfied` | enforced | An obligation the engine could not observe is reported as satisfied | pose-obligation-contract |
| `legacy-blocker-gains-an-actor` | enforced | A blocker known only as text is given an invented recipient | pose-obligation-contract |
| `blocked-counted-as-resolved` | enforced | Operational waiting counts as a resolved outcome in success metrics | pose-blocked-semantics-alignment |
| `flat-specs-share-one-journal` | enforced | Two flat specs record amendments in one shared journal | pose-flat-spec-amendments |
| `local-metadata-read-as-published` | enforced | A locally read version is presented as a proven publication | pose-public-claims-publication-provenance |
| `invented-trace-test-ref` | known-gap | A requirement trace cites a test that does not exist and `lint-spec --strict` passes (follow-up 097 of pose-abm-design-basis) | pose-mechanization-adversarial-corpus |
| `unavailable-producer-reads-as-empty` | enforced | A producer fails and Attention reports zero obligations as a complete answer | pose-obligation-projection |
| `trivial-change-raises-a-request` | planned | A trivial change or an already-authorized instruction produces an ActionRequest | pose-action-requests |
| `declined-approval-satisfies` | planned | A declined approval, or a cancellation without authority, releases a gate | pose-action-request-resolution |
| `closeout-plan-fills-judgment` | planned | The recoverable closeout plan answers a judgment criterion to finish | pose-recoverable-closeout-plan |
| `ready-from-absent-blockers` | planned | Absence of a known blocker is presented as proven independence | pose-phase-scoped-readiness |
