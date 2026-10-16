# Independent review: B02 approval domain preflight

Date: 2026-10-16 UTC. Decision: **changes requested**. Reviewer:
`agent:independent-gpt-6.1-sol-review`. Subject: PR #183, branch
`ccr-4ce0ffc2-alg4id`, head `4a59357be47fc903d660039ebfdc46584efbff23`,
spec `pose-review-approval-domain-preflight`, diff `1aec323a..HEAD`.

## Subject and authority

Native context resolved the spec to `proj.pose-dist`, same-project authority,
in-progress, without transfer or ambiguity. Implementation was by
`agent:claude-opus-5-5` (Claude/Anthropic); this review is another vendor's
execution (Codex/OpenAI). The maintainer authorized the reviewer id.
The `agent:independent-` prefix is a declaration required by the engine until
`pose-delegated-review-run-provenance` replaces it. Identity and separation
remain declared. No signature, verified separation or human confirmation is
claimed.

## Rules applied during review

- Change type: Go governance bugfix, consumer qualification and documentation.
- Read root AGENTS.md, POSE.md review/closeout contracts, review workflow,
  pose-review and pose-spec-closeout skills, R1–R7 and D1.
- Applied security (channel authentication and proof versus admission),
  backend-go (domain ownership and error propagation), documentation-style,
  delivery-evidence and knowledge-governance.
- Consulted the sealed-bundle decision log and action-answer historical-time
  knowledge. Frontend and infrastructure rules do not apply.
- No production, test, policy, capability or adoption edits were made.

## Behavioral evidence

Independently executed from pose-mcp:

```text
go test ./internal/pose ./internal/cli ./internal/mcpserver -run 'ReviewDomainPreflight|ReviewAssurance|ABMReviewSoundness|ABMReviewAuthority|ReviewAttest' -count=1
ok github.com/harne8/pose-mcp/internal/pose 0.473s
ok github.com/harne8/pose-mcp/internal/cli 0.362s
ok github.com/harne8/pose-mcp/internal/mcpserver 0.009s
```

Also independently executed the existing parser regressions:

```text
go test ./internal/cli -run 'TestAttestRejectsDispositionsItCannotStandBehind|TestReviewCriterionDispositionsRefuseJudgmentByOmission' -count=1 -v
PASS
ok github.com/harne8/pose-mcp/internal/cli 0.013s
```

Both pass, confirming that the separate CLI semantic rejection remains.
The supplied delivery-validation snapshot was inspected with jq using
`.checks[].outcome`: 133 passed of 133, no non-pass ids, outcome pass, git_head
equal to the reviewed head. This snapshot was supplied by the coordinator;
it is not represented as an independently repeated global validation.

## Requirement assessment

| Requirement | Observation |
|---|---|
| R1 | The writer completes identity then runs the shared verifier before creating the attestation directory. The admission implementation is materially improved; consumer convergence is incomplete as below. |
| R2 | Domain tests snapshot file content throughout .pose and assert unchanged refusals. The ordinary writer has no artifact writes before admission. CurrentReviewBundle prepares and matches existing bundles rather than sealing one. |
| R3 | Existing sealed gates feed validateBundleAttestation; signing/trust remain live. Historical fixtures explicitly bypass admission to exercise readers, with historical invalidity still asserted. |
| R4 | Domain negative decisions bypass approval semantics while preserving signed-channel and completion checks. The ordinary CLI still demands approval-oriented tools/judgments before building even a negative attestation; F1. |
| R5 | Focused tests cover valid canonical identity, unsigned/signed retry and stale/superseded input refusal. No general transactional/concurrency refactor is claimed. |
| R6 | Signature verification and semantic approval remain separate; signed invalid criteria disclose proof but cannot be recorded as approval. |
| R7 | Not satisfied: CLI helpers retain semantic policy and return before either shared preflight or writer; F1. |

## Judgment conclusions

- Assumption-integrity: no signed or human identity is inferred for this review.
  A trusted signature authenticates bytes, not the approval's merits.
- Compatibility: stricter write admission is reasonably classified minor.
  Consumer parity and negative-decision behavior need correction before approval.
- Design-causality: D1's shared admission addresses the original Store/import/MCP
  bypass. Leaving transport semantic validators prevents full realization of R7.
- Documentation: the claim that every channel receives the same blockers is
  broader than the inspected CLI implementation supports.
- Operability: exit 2 parser errors and domain typed refusal differ for the same
  semantic defect; operators cannot rely on the promised common diagnostics.
- Scope: legacy record without bundles is a separate writer/contract and its
  explicit exclusion is reasonable. Recomputing the current bundle is a bounded
  cost of stale-subject rejection; neither exclusion is the blocking finding.
- Security: observed domain ordering preserves authentication for negatives;
  no new credential or dependency exposure was identified in the reviewed diff.
- Solution-proportionality: reusing the verifier is cohesive, but convergence
  must include semantic helpers rather than only replacing the final preflight.

## Anti-mechanization and findings

**F1 — medium — CLI retains independent semantic admission, violating R7 and
restricting R4 negatives.** Location:
`pose-mcp/internal/cli/review_closeout.go:784`, with policy in lines 1321–1336
and 1357–1365; required-tool admission also occurs at lines 773–776.

Scenario A: on a current bundle with a required correctness criterion, submit
an approved attestation with `correctness|finding|f-nope|review finding` and no
recorded f-nope. The ordinary CLI calls reviewCriterionDispositions, which
rejects at line 1336, returning exit 2 at line 787. It never invokes the domain
preflight/writer and never renders ReviewApprovalRefusal. The same completed
attestation submitted to Store or signed import is rejected by domain admission
with its typed refusal and shared blockers. The parser's diagnostic differs
and can hide other blockers. The executed parser test's
`finding_naming_one_that_is_not_recorded` subtest confirms this path; the
executed domain missing-finding test confirms the domain counterpart.

Scenario B: a legitimate changes-requested or rejected decision on a plan with
a required judgment, without an answer to that approval judgment, is allowed
by the domain. The ordinary CLI cannot represent it without supplying the
judgment: the helper has no decision parameter and unconditionally errors on
unanswered judgments at line 1365. Even with tool dispositions supplied, it
returns before the writer. The executed omission test confirms this helper
behavior; the domain negative test explicitly records criteria=nil. The new
CLI negative test uses a fixture whose mechanical correctness criterion is
populated, so it does not exercise this boundary.

Expected action: retain CLI syntax decoding, route semantic admission through
the shared domain result, and cover orphan findings, multiple simultaneous
blockers and negative decisions with omitted approval judgments/tools across
ordinary CLI, signed import and Store. Preserve authentication and zero-write
refusal. No fix was applied by this reviewer.

The cheapest false confidence is testing parity only with an invalid accepted
risk that gets past the CLI builders. The new parity test does exactly that
input selection; it does not establish parity for semantic defects rejected
earlier by those builders. Historical direct-disk fixtures are appropriate for
historical readers and are not the cause of this gap.

## Tool dispositions

- Focused independent regressions and existing parser regressions: executed,
  passed, with the coverage limitation above.
- Context: resolved. Supplied strict validation: inspected, 133/133.
- Bundle explain: executed before index; reported no attributed change set and
  no attributed validation evidence in the existing index. No seal was made.
- Assess tech-debt: executed, zero markers. Assess integrate: executed, 64
  contracts, 63 static unobserved-consumer gaps. Those inventory gaps do not
  establish runtime failure of this change; HTTP MCP qualification passed.
- Index: initiated during review preparation, then interrupted with SIGINT
  (exit 130) after F1 blocked closeout. No index success is claimed.
- Remaining approval/closeout tools and final-head publication checks: not
  claimed as passed; closeout was stopped on F1 as explicitly requested.

## Decision

Changes requested. No sealed bundle or attestation was produced by this
review. Review verify, close, strict spec lint and strict check were not used
to advance the delivery. The spec remains in-progress. No closeout commit,
push, merge or upstream contribution was made. Remediation and a fresh
independent review are required before native approval and closeout.
