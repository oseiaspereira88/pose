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


## Second round

Date: 2026-10-16 UTC. Reviewed head:
`382708de9ff8be9bab48e70d3bfc817549ea7d3d`, diff `1aec323a..HEAD`.
Decision: **changes requested**, on F2 (medium). Reviewer:
`agent:independent-gpt-6.1-sol-review`, authorized by the maintainer.
This execution is Codex/OpenAI, a different vendor from the implementing
Claude/Anthropic execution. The `agent:independent-` prefix remains a declaration
required by the engine until `pose-delegated-review-run-provenance` replaces it;
there is no authenticated separation, signature or human confirmation claimed.

### Rules and scope

The first round's rule selection applies: security, backend-go,
documentation-style, delivery-evidence and knowledge-governance; the review and
spec-closeout skills and review workflow were read. Native context resolved
`proj.pose-dist` and the in-progress spec without ambiguity. Reviewed the domain,
CLI and MCP diff, the remediation and historical-fixture adaptations. The legacy
non-bundle record path is an explicit separate contract and its exclusion remains
reasonable. Recomputing the bundle is a bounded cost of checking current inputs;
no concurrency or transaction redesign is claimed.

### F1 verification and executed evidence

F1's criterion/tool decoder defect is resolved. Unknown and duplicate answers
are retained, omitted judgments and required tools remain omitted, and domain
validation rejects approval defects. Recommended-tool invalid deferred/not-used
rationales follow the existing domain warning policy; invalid dispositions,
unknown/duplicate tools and missing or unsuccessful required tools remain
blockers. No approval bypass from these decoder changes was found.

Independently executed on the reviewed head:

```text
go test ./internal/pose ./internal/cli ./internal/mcpserver -run 'ReviewDomainPreflight|ReviewAssurance|ABMReviewSoundness|ABMReviewAuthority|ReviewAttest' -count=1
ok github.com/harne8/pose-mcp/internal/pose 0.482s
ok github.com/harne8/pose-mcp/internal/cli 0.620s
ok github.com/harne8/pose-mcp/internal/mcpserver 0.009s
```

These include the new ordinary CLI semantic-refusal, signed-import multi-defect,
and omitted-negative-answers regressions. Historical disk fixtures intentionally
bypass admission to exercise readers; the helper completes canonical identity,
and reader tests still assert invalid history is refused without rewriting.
Other authority helpers assert typed refusal and an unchanged .pose tree rather
than silently treating a failed write as success.

An additional independent diagnostic used Go's `-overlay` with a temporary copy
of the test file, leaving repository production and test files untouched. It
used the existing CLI fixture, sealed a bundle, changed the validation matrix,
sealed its successor, and submitted the old bundle through CLI and Store. The
probe was stored under `.git/b02-review-probe/`, outside versioned artifacts.
Command from pose-mcp:

```text
go test -overlay ../.git/b02-review-probe/overlay.json ./internal/cli -run '^TestB02IndependentStaleTargetChannelParity$' -count=1 -v
DOMAIN APPROVAL BLOCKERS: [bundle rvb-397a0a601180e5e2 is superseded by rvb-9de787df33e917fe; review the current bundle missing required review tool disposition artifact-check missing required review tool disposition validate (component pose-mcp)]
CLI APPROVAL: exit=1 output=pose review attest: bundle is superseded by current semantic inputs
CLI NEGATIVE: exit=1 output=pose review attest: bundle is superseded by current semantic inputs
DOMAIN NEGATIVE: recorded decision=rejected bundle=rvb-397a0a601180e5e2
--- PASS: TestB02IndependentStaleTargetChannelParity (0.02s)
PASS
ok github.com/harne8/pose-mcp/internal/cli 0.028s
```

This diagnostic passes when it reproduces the discrepancy; it is evidence of
the defect, not an acceptance test pass. Fixture bundle ids are synthetic and
are not a seal or attestation of B02.

### Updated requirement assessment

| Requirement | Second-round assessment |
|---|---|
| R1 | Domain writer completes identity and runs verifier/time/current-subject admission before creating the artifact directory. No new approval bypass found in decoder remediation. |
| R2 | Domain refusal snapshots preserve file contents; CurrentReviewBundle prepares and matches without sealing or writing. CLI stale refusal writes nothing but loses the shared diagnostic, F2. |
| R3 | Sealed semantic gates and deliberately live trust/signing remain distinct. Explicit historical fixtures preserve reader coverage without authorizing new invalid approvals. |
| R4 | Fixed for omitted approval judgments/tools on a current bundle, with signed-channel checks retained. Still violated for a legitimate negative against a superseded bundle: ordinary CLI refuses what Store records, F2. |
| R5 | Domain rechecks current subject and preserves tested canonical retry identity. Both paths refuse stale approvals, but their diagnostics diverge, F2/R7. |
| R6 | Signature verification remains separate from semantic approval. Focused proof and signed-import regressions pass. |
| R7 | Not satisfied: CurrentReviewBundle and the explicit rvb target guard reject before domain admission and suppress the domain's full blockers, F2. |
| D1 | Domain distinction between identity, admission and approval is sound; the CLI's current-subject guard still imposes approval semantics on a negative. |

### Judgments and anti-mechanization

- Assumption-integrity: reviewer identity is declared, and signature verification
  is not evidence of semantic correctness or human confirmation.
- Compatibility: the documented exit 2 to 1 change is implemented for criterion
  and tool defects; the superseded-target discrepancy remains undocumented as a
  cross-channel limitation and contradicts the common-refusal claim.
- Design-causality: the domain is now the writer's admission point, but the
  ordinary CLI still owns a semantic freshness gate before reaching it.
- Documentation: POSE.md promises the same typed refusal and each verifier
  reason, including superseded bundles; the reproduced CLI output does not meet
  that promise.
- Operability: a stale target hides missing required-tool blockers; rejecting a
  review of the superseded subject requires a different channel or an unrelated
  fresh seal in the ordinary CLI.
- Scope: legacy record and bundle recomputation exclusions are acceptable;
  removing semantic divergence for supported bundle targets is within R4/R7.
- Security: inspected write ordering and signed-negative tests preserve channel
  authentication; no new credential exposure was identified.
- Solution-proportionality: the decoder correction is cohesive. Bundle selection
  must preserve an explicitly named sealed subject for domain judgment rather
  than preempting its decision.

The cheapest misleading pass is exercising only current-bundle inputs in the
parity corpus and checking merely that a superseded CLI approval fails. Existing
`TestReviewDomainPreflightCLIRefusesASupersededBundle` only asks for exit 1 and
`superseded`; it does not compare all domain blockers or try a negative on that
subject. The independent diagnostic exposes that missing boundary. No claim is
made that the existing adversarial corpus proves this case.

### Finding F2 — medium

Location: `pose-mcp/internal/cli/review_closeout.go:756` and `:764` (early returns
through line 766). Domain counterpart:
`pose-mcp/internal/pose/review_bundle.go:2425`–2440.

For an explicitly named sealed bundle B superseded by B2, ordinary CLI attest
loads the current bundle and exits with its own freshness message before
constructing the attestation or invoking shared admission. An approval missing
required tools therefore receives only the CLI message instead of the domain's
superseded and missing-tool blockers. A `rejected` decision on B receives the
same CLI refusal; Store records the same completed negative because current-
subject admission deliberately applies only to approvals. When no current seal
exists, the earlier nil-current return is another instance of this divergence.

Impact: R7's shared blockers are incomplete, and R4's legitimate negative audit
record is unavailable through the ordinary CLI for the review's actual sealed
subject. This is an existing guard left unconverged, not an approval bypass
introduced by the decoder remediation. F1 is resolved; F2 independently blocks
acceptance of the declared R4/R7 contract.

Expected action: resolve explicit bundle ids to the named immutable bundle and
let domain admission decide approval freshness; preserve authorized negative
recording and authentication. Keep syntax/explicit expected-digest checks as
appropriate, without replacing domain semantic blockers. Cover superseded and
stale-without-successor subjects, multi-defect approvals, and negatives across
ordinary CLI and Store/import/MCP. No code or test fix was made by this reviewer.

### Tool dispositions and decision

- Context and unsealed bundle explain executed. Explain reported missing indexed
  attributed change set and validation evidence; index/seal were not run.
- Assess tech-debt executed: zero markers, zero uncovered items.
- Assess integrate executed: 64 contracts, 63 static unobserved-consumer gaps.
  These inventory observations do not establish a runtime failure in B02;
  the focused HTTP MCP qualification passed.
- GitHub rate limit checked before validation: core 60 of 60 available.
- Global `pose validate --strict --json-out .pose/results/delivery-validation.json`
  started on 382708de, then interrupted with SIGINT (exit 130) after F2 was
  reproduced, following the instruction to stop on a medium finding. It reached
  pose-mcp's full Go tests and did not publish a new results snapshot. No current-
  head global pass or jq pass count is claimed; the committed snapshot predates
  remediation.
- Remaining approval/closeout checks were not represented as passed. No native
  seal, attestation, review verify, close, strict spec lint or strict check was
  used to advance the spec. No discovery state, contribution draft, code/test
  change, closeout commit, push or merge was made.

Decision: **changes requested**. Stop before index/seal/attestation/close; retain
in-progress lifecycle and roadmaps. A corrected consumer path and fresh review
are required before native closeout. The first round above remains intact.
