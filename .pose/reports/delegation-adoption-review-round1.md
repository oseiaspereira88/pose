# Delegated review capability — implementation review round 1

Source: separate Codex execution `20261029T021112-codex-895244`. Same-vendor limitations are preserved below. This is a preliminary source review, not a sealed approval.

**Changes requested: two high and two medium findings.** These are source-derived findings; I did not execute new regression tests.

I am OpenAI Codex, based on GPT-6; I cannot independently verify the exact `gpt-6.1-sol` runtime designation. This is a same-vendor separate execution, with correlated-model limitations—not cross-vendor independence.

1. **High — The protected baseline misses the newly introduced delegation floor.**
   [review_plan.go:272](../../pose-mcp/internal/pose/review_plan.go:272), [review_policy_baseline.go:245](../../pose-mcp/internal/pose/review_policy_baseline.go:245).
   **Trigger:** the committed baseline has `delegated_review: true`, ordinary independence set to same-actor, and no stronger overlay; the reviewed change disables delegation.
   **Consequence:** current planning removes the different-actor floor, while baseline restoration considers only `reviewer_independence` and profiles. The disabling change can therefore approve itself under same-actor review. The new protected-baseline test misses this because its fixture already declares `different-actor`.
   **Fix:** include the baseline’s adopted delegation floor when restoring independence. Add a regression where delegation alone supplies that floor. Preserve the maintainer’s live B17 proof-bar decision.

2. **High — MCP cannot reach the new unavailable-reviewer escalation.**
   [delegate_tools.go:366](../../pose-mcp/internal/mcpserver/delegate_tools.go:366), [delegate.go:478](../../pose-mcp/internal/cli/delegate.go:478).
   **Trigger:** call `pose_delegate_start` with `apply: true` when reviewers are unavailable, insufficiently independent, or the plan requires a person.
   **Consequence:** MCP first invokes dry-run, receives a refusal, and returns immediately. Dry-run deliberately creates no escalation, so R6/R11’s authorized action request never opens. Additionally, the MCP summary/refusal projections discard the new `escalation` field.
   **Fix:** provide an apply path for typed escalation outcomes while keeping previews mutation-free; preserve the validated request identifier through MCP. Test this through the MCP boundary, asserting one request and zero runs.

3. **Medium — Activation compares different bytes from those it validated.**
   [delegated_review_adoption.go:124](../../pose-mcp/internal/pose/delegated_review_adoption.go:124).
   **Trigger:** replace delegation configuration after `LoadDelegationPolicy`/activation validation but before the separate `before` read.
   **Consequence:** validation checks configuration A, while both byte comparisons see configuration B. Activation returns success even if B lacks the required issuer, provenance, or review agent. The existing race test replaces configuration later, during review-policy publication, so it misses this window.
   **Fix:** parse and validate the exact captured bytes used for the post-publication comparison. Add a deterministic replacement test between loading and comparison capture.

4. **Medium — Independence filtering stops routing before exhausting eligible agents.**
   [delegate.go:382](../../pose-mcp/internal/cli/delegate.go:382).
   **Trigger:** use supported `routing.prefer: "any"` with an available same-model agent first and an available independent agent later.
   **Consequence:** routing selects the first agent; B21’s subsequent independence check rejects it and escalates without considering the independent candidate. An otherwise autonomous review unnecessarily requires a person.
   **Fix:** incorporate required independence into candidate eligibility, retain skip explanations, and continue according to the operator’s configured ordering. Test same-model-first followed by a usable independent agent.

The cheapest formal shortcut is a trusted local command that discards the generated brief and emits a complete canned approval while declaring another vendor/model. Native signing can authenticate that executor declaration without proving substantive review occurred. Existing checks stop missing proofs, incomplete conclusions, task substitution, and several history manipulations; they do **not** establish reviewer cognition. The seeded-defect journey tests one cooperative fake reviewer, and the adversarial corpus has no B21 case documenting this shortcut. Record that boundary explicitly; do not describe these tests as preventing rubber-stamping.

I found the live B17 barrier preserved. Static inspection also supports explicit-refusal versus legacy-omission handling, machine-only init signing, retained recovery artifacts, and disclosure propagation. Distributed skills and translated copies matched; the English manual difference was the existing instance-owned contributor section.

Applied the review workflow and security, Go, documentation, delivery-evidence, and knowledge-governance rules. `pose context`, bundle preparation, source inspection, parity comparisons, and `git diff --check` were read-only. The prepared bundle was `rvb-a38a0a71f9e7383e`; it is **not sealed or attested**.

The tree changed during review, and earlier suite logs contain failures whose source fixes subsequently arrived. I therefore claim no final green validation. I made no edits, Git writes, attestations, agent launches, or vendor executions; broad suites and publishing assessment commands were left untouched.

## Author dispositions

- F01: restored the baseline delegated-review floor; regression `TestDelegatedReviewCapabilityProtectsTheAdoptedFloor`.
- F02: typed mutation-free escalation preview, authorized MCP apply and sanitized request id; regression `TestDelegatedReviewCapabilityEscalatesThroughMCP`.
- F03: parse and validate the captured bytes; deterministic replacement regression `TestDelegatedReviewCapabilityValidatesTheCapturedConfiguration`.
- F04: mark incompatible candidates ineligible before routing, retain skip reasons and continue; regression `TestDelegatedReviewCapabilityRoutesPastASameModelAgent`.
- Trust boundary: added the adversarial disclosure case `canned-executor-approval-proves-cognition`; no claim to detect canned judgments.

Focused regressions passed. Final registered validation and a new review remain necessary after these source changes.
