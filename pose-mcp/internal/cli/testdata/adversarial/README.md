# Adversarial corpus

Spec: `pose-mechanization-adversarial-corpus`. Test: `adversarial_corpus_test.go`.

Every gate POSE adds against mechanical compliance can itself be satisfied
mechanically. Each row is the cheapest formal satisfaction of one gate. An
`enforced` case fails the build when the shortcut works again; a `known-gap`
case fails the build when the gap closes, so it cannot close or reopen
silently; a `planned` case belongs to a spec not yet delivered and gains code
before that spec closes; a `delegated` case is enforced by a test in another
package, named in the row, whose existence the corpus checks.

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
| `trivial-change-raises-a-request` | enforced | A trivial change or an already-authorized instruction produces an ActionRequest | pose-action-requests |
| `declined-approval-satisfies` | enforced | A declined approval, or a cancellation without authority, releases a gate | pose-action-request-resolution |
| `closeout-plan-fills-judgment` | enforced | The recoverable closeout plan answers a judgment criterion to finish | pose-recoverable-closeout-plan |
| `ready-from-absent-blockers` | enforced | Absence of a known blocker is presented as proven independence | pose-phase-scoped-readiness |
| `stale-preview-authorizes-apply` | enforced | A start previewed before a restricting request is applied after it | pose-governed-effect-enforcement |
| `code-change-keeps-judgment` | enforced | The reviewed code changes and every criterion still reads as equivalent, so the old judgment is reused | pose-material-equivalence-reuse |
| `minimal-surface-skips-trace` | enforced | A done spec declares the minimal surface to drop its requirement trace | pose-progressive-spec-surface |
| `stale-assessment-reused` | enforced | The assessed component changes and the old assessment still reads as fresh | pose-adaptive-assessment-freshness |
| `unpinned-premise-trigger-reads-current` | enforced | A premise trigger without a pin can never go stale and nothing says so | pose-assumption-validity-scope |
| `calendar-ttl-as-premise-trigger` | enforced | An arbitrary expiry date is accepted as the material trigger of a premise | pose-assumption-validity-scope |
| `unobserved-falsifier-reads-uncontradicted` | known-gap | A falsifier check that no run produces raises nothing, and nothing says the falsifier was never observed | pose-falsifier-reconsideration |
| `parity-row-cites-an-unrelated-test` | known-gap | A launcher parity row names a real but unrelated `pose delegate` command and an existing test that does not exercise the capability, and the parity check passes; the check proves existence, review of the cited tests proves coverage | pose-delegation-parity-gate |
| `imported-fragment-completes-a-replaced-command` | known-gap | A usage or error string assembles a removed command from a constant of another package of the module, which the active-document check reads as a gap; review of assembled messages is the defence | pose-delegation-parity-gate |
| `cgo-preamble-text-reaches-usage` | known-gap | Operator text written as a C string in a cgo preamble reaches a usage surface; the Go reader sees a comment, not a string | pose-delegation-parity-gate |
| `request-opened-after-preview-transfers` | delegated | A request opened after a transfer preview is moved with the spec (`pose/spec_transfer_actions_test.go:TestTransferRefusesARequestOpenedAfterThePreview`) | pose-transfer-preserves-obligations |

| `deleted-reservation-resets-quota` | delegated | Deleting a reservation and receipt makes a surviving run stop consuming quota | pose-delegated-review-attempt-ledger |
| `witness-bundle-filter-hides-loss` | delegated | Editing a witness bundle makes scoped readers skip its corruption | pose-delegated-review-attempt-ledger |
| `window-erases-rerun` | delegated | Moving a prior run outside the time window turns a rerun into a first attempt | pose-delegated-review-attempt-ledger |
| `canned-executor-approval-proves-cognition` | delegated | A complete canned approval from a trusted local command is presented as proof of review cognition or a vendor receipt (`cli/delegated_review_capability_test.go:TestDelegatedReviewCapabilityCannedApprovalDisclosesItsLimits` guards disclosure; the engine cannot detect canned judgments) | pose-delegated-review-capability |
| `oversized-writer-publishes-unreadable-run` | delegated | A writer reports completion for a run its bounded reader immediately refuses | pose-delegated-review-attempt-ledger |
