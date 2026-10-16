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


## Third round

Date: 2026-10-16 UTC. Reviewed head:
`c87a7ffc0da64f857980f5b00fe7abff8d511248`, diff `1aec323a..HEAD`, PR #183,
branch `ccr-4ce0ffc2-alg4id`. Decision: **changes requested**, on F3 and F4
(medium), reported together after inspecting the complete admission paths.
Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer.

### Authority, rules and judgments

Native context resolved the in-progress spec to same-project authority
`proj.pose-dist`, without ambiguity or transfer. Read AGENTS.md, POSE.md's
review/closeout contract, review workflow, pose-review and pose-spec-closeout
skills, the spec, production diff, changed regression fixtures and applicable
security, backend-go, documentation-style, delivery-evidence and knowledge rules.
Consulted the sealed-review-bundles and action-answer historical-time decision
logs. No frontend, deployment or infrastructure changes require those rules.

- Assumption-integrity: the prefix `agent:independent-` is a declaration required
  by the engine until `pose-delegated-review-run-provenance` replaces it. This is
  a different vendor's execution, Codex/OpenAI versus the implementing
  Claude/Anthropic (`agent:claude-opus-5-5`). Identity and independence remain
  declared; no signature, authenticated separation or human confirmation is
  claimed. The coordinating Claude session's validation is not this review.
- Compatibility: refusing writes the verifier already rejected is appropriately
  classified minor. The documented criterion/tool exit 2 to 1 remediation works;
  the remaining mapping and auto-attest divergences prevent the universal
  consumer-parity claim from being true.
- Design-causality: completing canonical identity and reusing the verifier before
  opening the attestation directory implements D1 in the writer. Transport
  mapping acceptance and auto-attest freshness still preempt that result.
- Documentation: the scope and residual-cost disclosures are useful. The claim
  that CLI surfaces consume the shared blockers needs the remaining convergence.
- Operability: mapping failures hide other actionable blockers; auto-attest's
  preparation-only success does not disclose approval-admission failure, and its
  stale guard hides the full domain result.
- Scope: excluding legacy non-bundle `review record` is acceptable: it is a
  separate attempt contract/writer. Recomputing CurrentReviewBundle is the
  bounded cost of enforcing current semantic inputs, with no general
  transaction/concurrency redesign claimed. Neither is a new finding.
- Security: negative records still require their channel's signature/admissibility;
  a signature authenticates bytes rather than approval. No new persistence bypass
  was found in the F1/F2 fixes. No credentials or human confirmation were invented.
- Solution-proportionality: sharing the verifier is cohesive. Finishing the
  remaining decoders/consumer projections is within B02, without rewriting writers
  or introducing new policy.

### F1 and F2 verification; independently executed regressions

F1 is resolved for criterion/tool answers: unknown and duplicate answers survive
syntax decoding; omitted judgment and required-tool answers stay omitted; domain
admission reports their blockers together. Negatives do not owe those approval
answers. Recommended-tool warning semantics are unchanged in the domain; invalid
or missing required-tool dispositions remain blockers. No new approval bypass
was found from removing those parser checks.

F2 is resolved in `cmdReviewAttest`: an explicit bundle id loads the named
immutable bundle. The domain reports supersession with other blockers for an
approval, and the CLI can record a negative about that old bundle. A scope target
continues to select its current seal. F4 below concerns the separate auto-attest
consumer, not a failure of the new resolver's ordinary-attest regression.

Executed from `pose-mcp`:

```text
go test ./internal/pose ./internal/cli ./internal/mcpserver -run 'ReviewDomainPreflight|ReviewAssurance|ABMReviewSoundness|ABMReviewAuthority|ReviewAttest' -count=1
ok github.com/harne8/pose-mcp/internal/pose 0.513s
ok github.com/harne8/pose-mcp/internal/cli 0.519s
ok github.com/harne8/pose-mcp/internal/mcpserver 0.010s
```

Also executed the following with `-count=1 -v`; all passed, CLI package 0.216s:

```text
TestReviewDomainPreflightCLISemanticRefusalsComeFromTheDomain (all nine subcases)
TestReviewDomainPreflightSignedImportAgreesOnSeveralDefects
TestReviewDomainPreflightNegativesOweNoApprovalAnswers (both subcases)
TestReviewDomainPreflightCLISupersededBundleIsTheDomainsToJudge
```

Historical disk fixtures complete canonical identity and explicitly model
pre-admission history. Their reader assertions still reject invalid history;
other adapted authority tests assert typed refusal and unchanged .pose content.
This does not silently authorize new invalid production approvals. The remaining
coverage gaps are transport mapping and auto-attest admission, not those helpers.

### Updated requirement assessment

| Requirement | Third-round assessment |
|---|---|
| R1 | Writer completes identity then validates sealed semantics, current trust, operation time and current subject before artifact creation. No new writer bypass found. |
| R2 | Domain refusal snapshots preserve existing artifacts. CurrentReviewBundle prepares and matches existing seals without creating a bundle; no record, supplement or approval event is written before refusal. Consumer diagnostics remain incomplete under F3/F4. |
| R3 | Semantic gates come from the seal; deliberately live signing/trust and historical-reader checks remain distinct. Historical fixtures preserve read-side coverage. |
| R4 | F1/F2 negatives pass, including omitted approval answers and superseded ordinary-attest targets. A negative carrying a reasonless mapping is still refused solely by the CLI, F3. Signed-channel proof is retained. |
| R5 | Valid approval, canonical unsigned/signed retry and stale-input refusal regressions pass. Ordinary-attest preview/apply use domain admission; auto-attest preview omits it, F4. No transactional race guarantee beyond the scoped design is claimed. |
| R6 | Signed invalid semantics can disclose identity proof but cannot persist an approval. Signed import and HTTP MCP regressions pass. |
| R7 | Not satisfied: mapping rationale acceptance and auto-attest freshness/preview remain CLI-only behavior, F3/F4. |
| D1 | Domain separation of identity proof, write admission and approval semantics is correct; transport convergence remains incomplete. |

### Complete pre-domain guard inventory

This inventory follows ordinary attest, signed/envelope import and MCP to the
shared writer, and checks the adjacent auto-attest route explicitly named in the
spec. It distinguishes representational/channel checks from approval policy.

| Guard or branch | Judgment |
|---|---|
| CLI missing option values, required target/reviewer/decision/evidence inputs, unexpected arguments, pipe field counts and slug decoding | Acceptable command grammar/minimum request construction; no plan-specific approval conclusion. Findings decode fields rather than deciding severity/risk acceptance. |
| Bundle path traversal checks, LoadReviewBundle and scope-to-current-seal selection | Acceptable addressing/integrity. Explicit ordinary-attest ids now preserve the named subject; no supersession verdict is imposed there. |
| Explicit `--plan-digest` mismatch | Acceptable caller-requested expected-value guard; not implicit approval policy and not required by another channel. |
| `--mapping` criterion owner and observed-fact addressing | Acceptable bounded flag addressing, as explicitly disclosed: mappings must identify the plan's structural obligation and observed fact. This is distinct from judging an answer's justification. |
| Mapping field arity, owner/date form restricted to accepted-risk, requiring a basis ref instead of the literal mapped token | Acceptable encoding of the documented alternative flag forms. |
| Mapping missing-evidence/not-applicable/accepted-risk requires nonempty rationale | Semantic approval acceptance, not decoding: all fields can be represented, the domain already checks this for approval and skips it for negatives. F3. |
| Attribution schema/role/mode/confirmation binding | Shared domain ValidateReviewAttribution, also called by canonical completion; an internally inconsistent declaration is invalid for negatives too. Acceptable integrity check, without treating a declared principal as proof. |
| `--authority` needs `--sign`; native key load; `--review-execution`; positive duration parsing | Explicit request to construct an authenticated claim, its required fields and supported duration representation. Acceptable channel/construction guards; actual grants, authority separation, trust and approval are evaluated by the domain. No issuer key or review artifact is created by key loading/signing. |
| CLI signed import/envelope verification before preview/apply | Shared domain verification authenticates the channel; preview uses PreflightReviewAttestation and apply uses the writer. Negative decisions retain proof checks. |
| MCP JSON decoding, required draft/envelope, apply requires trusted envelope; server project/access governance | Acceptable transport/authentication boundary. SignableReviewAttestation canonicalizes the draft; envelope/draft preview reports domain blockers; apply invokes RecordReviewAttestationEnvelope. The typed refusal is mapped to review_approval_refused with recorded=false. |
| Domain policy/signature requirement, canonical bundle digest, reviewer/decision/time/id/attribution validation | Shared request integrity/authentication before semantic admission, not a second CLI/MCP approval policy. |
| Auto-attest explicit-id supersession and no-current-seal guard | Implicit CLI freshness admission for a named subject, before domain preparation/admission; drops other blockers. F4. |
| Auto-attest incomplete preparation, pending judgments | Acceptable shared preparation contract (the Store auto-attest route also refuses to invent judgments); does not authorize approval. |
| Auto-attest complete preview exits before PreflightReviewAttestation | Preparation does not evaluate admission. Current preview and apply can disagree on unchanged inputs, F4. |

No additional medium-or-higher finding was identified in this inventory. In
particular, recommended-tool warnings do not establish a new required-tool
approval bypass, and neither attribution nor signed-channel guards dispense
with proof on negatives.

### Finding F3 — medium — mapping acceptance still lives in CLI

Location: `pose-mcp/internal/cli/review_closeout.go:996`–997; early return
at :812–815. Domain counterpart: `pose-mcp/internal/pose/review_structure.go:310`–312,
inside approval validation.

On a sealed plan with a structural-mapping criterion and an observed material
fact, submit `--mapping CRITERION|DELTA|missing-evidence|` together with an
approval missing required tools. This is decodable: it names a criterion, a real
fact and a known disposition, with an empty rationale. CLI returns its own exit
2 message before constructing the attestation. Store returns ReviewApprovalRefusal
with both the missing rationale and missing-tool blockers. Changing only the
decision to rejected records through Store, while CLI still refuses the same
mapping in preview and apply. The same branch covers not-applicable and
accepted-risk with empty rationale. This is the same substantive R4/R7 parity
requirement as F1; it is not the disclosed mapping-addressing exception.

Expected action: decode these dispositions and preserve the empty rationale for
the domain to judge, retaining genuine addressing/representation guards. Cover
all three dispositions, multi-defect approval and authorized negatives through
CLI and domain/signed routes. No production/test fix was made by this reviewer.

Independent diagnostic used a Go overlay stored only under
`.git/b02-third-round-probe/`; versioned production/tests remained untouched. It
created a real synthetic Git dependency delta, selected a structural plan,
sealed it, and exercised the complete command and Store:

```text
go test -overlay ../.git/b02-third-round-probe/overlay.json ./internal/cli -run '^TestB02ThirdRoundMappingAdmission$' -count=1 -v
DOMAIN APPROVAL BLOCKERS: [criterion correctness disposes SD-4b7a5faa as missing-evidence with no rationale missing required review tool disposition artifact-check missing required review tool disposition validate (component pose-mcp)]
CLI approved apply=false exit=2: mapping SD-4b7a5faa for correctness is missing-evidence and needs a rationale
CLI approved apply=true exit=2: same message
DOMAIN NEGATIVE recorded: rejected
CLI rejected apply=false exit=2: same message
CLI rejected apply=true exit=2: same message
--- PASS: TestB02ThirdRoundMappingAdmission (0.03s)
ok github.com/harne8/pose-mcp/internal/cli 0.036s
```

The diagnostic passes on reproducing the defect, not on satisfying acceptance.
Its synthetic ids are not B02 approval artifacts.

### Finding F4 — medium — auto-attest omits shared admission on preview and stale targets

Location: `pose-mcp/internal/cli/review_closeout.go:1049`–1059 and :1063–1069.
The domain writer is reached for a complete apply at :1074, but not for those
paths. The spec's Tasks and Approach explicitly include auto-attest convergence.

With a complete mechanical plan and different-actor independence, auto-attest
preview using `agent:reviewer` exits 0 with complete=true and no admission blocker.
On unchanged inputs its apply receives the domain refusal for reviewer
independence. After sealing a successor bundle, auto-attest of the old explicit
id exits at its CLI supersession guard. The domain's same prepared attestation
has both supersession and independence blockers, but neither preview nor apply
reports that complete result. F2 fixed ordinary attest, leaving this adjacent
copy of the freshness policy in place. No invalid approval is persisted: apply's
writer admission works; the defect is incomplete R7 convergence and preview
admission disclosure.

Expected action: preserve an explicitly named subject for auto preparation and
project the shared admission result when presenting a complete approving draft;
apply must retain writer revalidation. Keep incomplete judgments pending rather
than fabricating them. Cover unchanged-input preview/apply refusal parity and
superseded/stale targets with multiple blockers. No fix was made here.

Executed independent overlay diagnostic:

```text
go test -overlay ../.git/b02-third-round-probe/overlay.json ./internal/cli -run '^TestB02ThirdRoundAutoAttestAdmission$' -count=1 -v
DOMAIN CURRENT BLOCKERS: [review policy requires a different-actor reviewer identity; under declared assurance that is read from the reviewer prefix, not authenticated]
AUTO CURRENT apply=false exit=0: pending_count=0 complete=true apply=false
AUTO CURRENT apply=true exit=1: domain different-actor refusal, nothing written
DOMAIN OLD BLOCKERS: [bundle rvb-c8c09ea58ae3a634 is superseded by rvb-457c1b0199f62b47; review the current bundle review policy requires a different-actor reviewer identity; under declared assurance that is read from the reviewer prefix, not authenticated]
AUTO OLD apply=false exit=1: bundle rvb-c8c09ea58ae3a634 is superseded by rvb-457c1b0199f62b47
AUTO OLD apply=true exit=1: same CLI-only supersession message
--- PASS: TestB02ThirdRoundAutoAttestAdmission (0.03s)
ok github.com/harne8/pose-mcp/internal/cli 0.037s
```

Again, a diagnostic reproduction pass is evidence of a defect, not approval.

### Anti-mechanization

The cheapest formal pass is to exercise only criterion/tool flags and ordinary
attest, then assume the word "domain" in the writer proves every consumer
converges. F3 uses valid addressing to reach semantic policy left in a neighboring
mapping helper; F4 exercises the adjacent auto-attest copy of the stale guard
and unchanged-input preview. The existing focused suite passes despite both.
These diagnostics expose gaps; no claim is made that the checked-in adversarial
corpus already proves them. Historical direct-disk fixtures are not the cause.

### Tool dispositions and decision

- Native context, unsealed bundle explain, focused independent tests and the
  explicit F1/F2 regressions executed. Explain returned prepared/unsealed with
  15 criteria, 11 tools, no attributed change set and no collected validation
  evidence in the existing index; it did not seal B02.
- Both independent Go-overlay diagnostics executed and reproduced F3/F4.
  Probe sources remain untracked inside .git, outside production/test artifacts.
- Assess tech-debt executed: zero markers and zero uncovered items.
- Assess integrate executed: 64 contracts, 63 static unobserved-consumer gaps,
  matching prior-round counts. The scanner's absence-of-reference inventory is
  not evidence of a runtime B02 regression; the real HTTP MCP regression passes.
- Suggest review for pose-mcp executed. Recurrence-check executed: zero flagged
  keys, zero flapping keys, one previously resolved cluster, 393 records scanned.
- Assessment-generated changes to integrations/technical-debt report/state files
  were restored after inspecting git show --stat; they are not delivery evidence
  or changes to commit. No discover --update-state output was committed.
- Global strict validation and jq pass-count publication were not performed in
  this round: a medium finding stops delivery before index/seal under the user's
  explicit instruction. The committed delivery snapshot predates c87a7ffc and
  is not claimed as current-head validation. No environmental failure is used
  as a substitute for a test result.
- Artifact/surface/history/knowledge closeout checks, final-head CI/Security,
  index, seal, attestation, review verify, close, lint-spec --strict and check
  --strict were not used to advance delivery and are not claimed as passed.

Decision: **changes requested**, F1/F2 resolved, F3/F4 medium outstanding. No
native sealed B02 bundle, attestation or closeout was produced; verify/close/lint/
check are not executed for promotion. No closeout commit or push was made, and
no PR merge, contribution staging, policy/capability/adoption or code/test edit
was performed. Only this Third round section was appended to the existing
report. Keep the spec in-progress and reconcile neither roadmap nor delivery
completion until these findings are remediated and independently reviewed.
