# B01 disclosure qualification — 2026-10-15

## Scope and reconciliation

Base main: `1f3244614d56f377d6d17b7e19ba5dd24e0285ac` (#181). #178 published
B04's native closeout; #179 closed action-answer. #181 published its final
verification and exact Security false-positive diagnosis. B01 remains partial
until the existing `pose-review-disclosure-requires-proof` reaches native
closeout. B02 is next on the preferred DAG chain, after merge; independent
roots do not acquire artificial dependencies. Administrative authoring closeouts
and conditional B43 are not functional implementation.

This delivery reuses the #158 implementation in `verifiedAuthorityBlockers`.
No new authority, policy, migration, release or persistent service is introduced.
R1–R4/D1 and the original `pose start` record remain unchanged. Native start
status classifies every node as recorded-before-managed-execution; no second
start is applied. The start preview reports no dependency wait and refuses a
new start because the spec is already in-progress, as expected.

## Qualification

Built the current source with Go 1.26.9. `assess discover --if-stale --component
pose-mcp`, `assess integrate`, `assess tech-debt` and `assess design --spec
pose-review-disclosure-requires-proof` executed before review. Related knowledge:
`knowledge:review-assurance-requires-cryptographic-proof` and
`knowledge:action-answer-proof-and-historical-time`. Their original dated
limitations remain historical; B04 now owns the implemented temporal/reuse
contract, with live pins/grants and no historical authorization receipt.

Executed domain regressions passed: invalid/malformed/missing signatures,
unsigned decision/claim changes, wrong subject/algorithm, untrusted key/revoked
pin, supplied-bundle mismatch, valid identity with invalid approval semantics,
all shared gate consumers, and invalid human confirmation. Renamed the last
test to `TestReviewAssuranceInvalidProofCannotVerifyHumanConfirmation` so the
existing `ReviewAssurance` integration selector includes it.

Added `TestReviewAssuranceProofCLIAndMCPConsumers`: signed positive control,
damaged signature, missing proof and valid signature with missing criteria.
It calls the CLI verify/review-check/closeout-check in JSON and human form,
and the real HTTP JSON-RPC `pose_review_bundle` and `pose_closeout_state` tools.
All wire assurance fields match the domain result; valid identity remains
verified when semantic criteria prevent approval. Empty omitted slices are
normalized by JSON round-trip for comparison. Directly seeded synthetic records
exercise the read path independently of approving-write admission (B02).
The integration test passed with `go test ./internal/cli -run
TestReviewAssuranceProofCLIAndMCPConsumers -count=1`.

Targeted race and current global native strict validation results are
recorded below. Historical failed validation from #158 remains
preserved and is not current evidence. CI and Security must finish on the final
published head.

## Rules applied during review

- Change type: qualification of a Go security bugfix, integration tests and governance documentation.
- Workflows: `.pose/workflows/feature.md`, `.pose/workflows/review.md`.
- Security: authenticate signed bytes and current issuer trust before disclosing identity; include negative tests without exposing credentials.
- Delivery evidence: require native strict execution, artifact provenance and composed-capability evidence.
- Documentation style and knowledge governance: preserve original baseline/history and distinguish implementation from formal closeout.
- Backend-go extension is not installed; `pose suggest feature --path pose-mcp` selects security/documentation-style. No unrelated extension adoption.
- Frontend, Kubernetes and release publication rules are outside this scope.

## Review and closeout

Initial qualification required current validation and different-actor judgments;
the native admission section below records their completion. No mechanical
preview constitutes approval. Identity/separation are declarations unless the
native verifier proves otherwise. No maintainer conclusion is inferred from
authorization to implement. B01 delivery_complete remains false until both
specs are terminal; no next unit starts before this PR merges.


## Initial native execution and independent findings

Global `pose validate --strict --json-out .pose/results/delivery-validation.json`
passed 132/132 in 298.6s, with no failures/errors/skips. This first run began on
local precursor `e4cde934`; its tree is identical to published `fa47ecc8`, but its
provenance is retained as that precursor and is not relabelled. After the
review/documentation candidate is published, regenerate current native results
before sealing. `go test ./...` passed in both Go modules; the matrix executed
all declared integration/reachability checks, including the new CLI/MCP test and
renamed human-confirmation regression through its existing selector.

Targeted `go test -race ./internal/pose ./internal/cli ./internal/mcpserver -run
'ReviewAssurance|ReviewConsumersAgree|ReviewTrustTime|ABMReviewAuthority|ReviewTrustReuse'
-count=1` passed: domain 2.469s, CLI 2.751s; MCP package reported no matching tests.
The actual MCP HTTP interactions run inside the CLI integration test under the
race detector; no separate MCP-package race coverage is claimed.

`agent:independent-codex-pr182` reviewed the original fix and current qualification
in a separate agent execution and ran its own focused behavioral tests. The
[review report](2026-10-15-pose-b01-disclosure-review.md) answers all eight judgments.
No code/security blocker was found. Its current-section B04 wording finding was
corrected; historical constraints and D1 remain unchanged. Registered corpus
coverage is distinguished from focused regressions; no absent corpus case is
claimed. The native plan and `state --attention` require fresh reviewer approval;
no maintainer acceptance action is owed for this spec. The previous action-answer
acceptance is neither reused nor fabricated.


## Native admission and terminal closeout

Current native global strict passed 132/132 in 241.1s on published
`ea53653b7f0aeebb8a22d394b9be3ce39eb8578f` (zero failed, optional_failed,
errored or skipped). Of these,129 checks cover pose-mcp and form the sealed
component evidence; the other3 cover mcp-enforce. Native artifact-check strict
had zero errors; surface-check strict had1 composed-capability target,
132 validation results and zero findings. The versioned execution summary is
`b01-disclosure-surface-check-summary.json`; authoritative input remains
`delivery-validation.json`, with native compact delivery index and sealed
bundle. The summary explicitly is not a substitute validation run. The full
expanded surface JSON is an intermediate; its exact digest/size are recorded
without adding a redundant21MB graph to the repository.

Native index ran before seal. Current assess-design observed8/8 attributed files
and zero material structural deltas. Independent admission sealed
`rvb-3bdcc9c37e034dc4`, plan
`sha256:10eb56a14b9c3261d4a89b17b8af4dd109069f9c9ec0ecad69c20e5428646cda`,
and recorded `rva-3a33dadbf95db8b8`:7 mechanical criteria plus8 explicit causal
judgments. Verify was fresh/approved/ready-to-close; review-check had no blockers.
The reviewer prepared, concluded and applied that record; confirmation is none,
identity/separation are declared and cognition is not observable. There was no
new maintainer acceptance request, and #179's acceptance was not reused.

`pose close spec:pose-review-disclosure-requires-proof --apply --reviewer
agent:independent-codex-pr182` applied the guarded transition, including the
native delivery gate. It emitted `closeout_plan.step.transition=done` and
`closeout_plan.result=closed`. `b01-disclosure-closeout-admission.json` reports
terminal=true, lifecycle_done=true, review approved and blockers=[]. Subsequent
native review verification reported fresh=true, approved=true, state=closed.
The lifecycle/date were written by the engine. Original start and R1–R4/D1
remain unchanged; start-status retains all5 recorded-before-managed-execution
nodes. The completed B04 follow-up is dispositioned done, with its historical
trust boundary preserved. Native discovery/state was refreshed on closure;
the existing scripts assessment is included without dropping that component.

Candidate CI38815135675 and Security38815135587 both passed on ea53653b.
Publishing lifecycle and current summaries adds metadata to the attributed
subject; the independent reviewer rechecks/reaffirms the final published subject
before readiness. Latest IDs/state are taken from
`b01-disclosure-review-verification.json` and `b01-disclosure-closeout-check.json`,
not inferred from the done flag or the closeout plan. The final PR head's CI and
Security remain required; their immutable links are recorded in PR #182.

B01 now has both reused specs closed natively and delivery_complete=true.
B02 is the next preferred technical unit (`pose-review-approval-domain-preflight`)
after this PR merges. B03 also depends on B01; other roots remain independent.
No new unit, policy adoption, release or conditional B43 work is started here.
