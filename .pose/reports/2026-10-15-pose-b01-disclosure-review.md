# Independent review: B01 review disclosure qualification

Date: 2026-10-15. Reviewer: `agent:independent-codex-pr182`, a separate agent
execution from the implementing agent. Scope:
`spec:pose-review-disclosure-requires-proof`, PR #182. This report records
independent source and behavioral review and the approved native closeout
admission. Current native strict validation passed, the separate reviewer
recorded the attestation, and the guarded native close completed. Final published
state verification remains required; this report alone is not its evidence.

## Subject and authority

At review start, the local context resolver selected `proj.pose-dist`, the existing
in-progress spec and its canonical authority. I inspected R1–R4, D1, the original start and
implementation commit `164c2913`, the current `review_bundle.go`, and qualification
commit `fa47ecc8`. The original cryptographic fix came from merged #158; this PR
adds CLI/MCP qualification and includes the existing human-confirmation regression
in the `ReviewAssurance` integration selector. B04 was delivered by #178;
action-answer was closed in #179 and its publication complement merged in #181.
The implementation matrix kept B01 partial until this sibling's native closeout.
The terminal admission below supports its completion summary; B02 still waits
for the current delivery to merge.

The effective plan selects `spec-closeout@1`, `backend-review@1`,
`engineering-judgment@1` and `high-criticality-review@1`: fifteen criteria,
including eight judgment criteria, critical band, different-actor independence.
Admission used the sealed bundle and effective plan from the current native
run, identified below. Final publication may supersede the bundle when attributed
governance content changes; consume the final native result files for its identity.

## Rules applied during review

- Change type: qualification of an existing Go security bugfix, tests and
  operational governance documentation.
- Read `AGENTS.md`, relevant `POSE.md` review/closeout contracts,
  `.pose/workflows/review.md`, `pose-review`, `pose-feature` step 10 and
  `pose-spec-closeout`.
- Security: authenticate signed bytes under current issuer pins before
  describing authority; inspect positive and negative authorization cases.
- Documentation style and delivery evidence: distinguish implementation,
  behavioral qualification and native terminal closeout; preserve dated history.
- Knowledge governance: inspect the existing dated decision log and
  `knowledge:action-answer-proof-and-historical-time`; no new knowledge artifact
  or accepted-risk handoff is needed for an unresolved finding.
- `pose suggest review --path pose-mcp` selects security/documentation-style.
  The resolved plan names backend-go; its extension file is absent in this
  instance. I evaluated Go contracts, errors, lifecycle and integration directly;
  no extension adoption or policy change is implied.
- Frontend, Kubernetes, infrastructure deployment and release rules do not apply.

## Behavioral evidence and requirement review

I independently executed, with Go 1.26.9:

```text
go test ./internal/pose ./internal/cli -run 'TestReview(Assurance|ConsumersAgreeOnTamperedProof|TrustTime|SurfacesDiscloseDeclared|ABM)' -count=1
ok github.com/harne8/pose-mcp/internal/pose 0.580s
ok github.com/harne8/pose-mcp/internal/cli 0.984s
```

The separately executed selector `TestABMReviewAuthority|TestReviewSurfacesDiscloseDeclaredAssuranceOfAHumanLabel`
also passed (`internal/pose` 0.276s, `internal/cli` 0.099s), covering the existing
authority-contract cases and declared human-label rendering explicitly.

This is focused behavioral evidence, not a substitute for the registered global
native strict run. The implementer also reports targeted race success for the
pose and CLI packages. Its mcpserver package invocation selected no tests, so no
MCP-package race coverage is claimed; the real HTTP MCP consumer is exercised by
the CLI integration test and its CLI race run.

| Requirement | Source observation and behavioral evidence |
|---|---|
| R1 | `verifiedAuthorityBlockers` calls the stored-envelope verifier before producing a verified authority result. `TestReviewAssuranceRejectsInvalidCryptographicProof` starts from a verified signed positive control, then changes malformed/damaged/missing signature, unsigned decision/claim, wrong subject, unsupported algorithm, untrusted key or revoked pin. Disclosure, JSON and human rendering refuse verified identity/claim/separation. |
| R2 | The shared checker compares both supplied bundle ID and digest, then retains claim digest, issuer, principal, audience/project, role/grant and execution checks. `TestReviewAssuranceBindsTheSuppliedBundle` rejects another supplied ID even with the copied digest; the existing ABM authority suite covers claim and separation boundaries. |
| R3 | Approval criteria and decision stay in `validateBundleAttestationWith`, outside authority proof. `TestReviewAssuranceKeepsIdentitySeparateFromApproval` resigns semantically invalid criteria/decision and retains verified identity while approval fails. The new CLI/MCP test reproduces this through public consumers. Declared legacy identity remains declared. |
| R4 | Domain verify, review-check, closeout and attribution consume the same assurance/proof path. `TestReviewConsumersAgreeOnTamperedProof` checks domain parity; `TestReviewAssuranceInvalidProofCannotVerifyHumanConfirmation` starts with a verified human confirmation, damages its signature and retains only declared confirmation. `TestReviewAssuranceProofCLIAndMCPConsumers` compares all wire assurance fields for signed positive, damaged, missing and invalid-criteria records across CLI JSON/text and real HTTP JSON-RPC tools `pose_review_bundle` and `pose_closeout_state`. |

Tests seed corrupt stored records directly, because an approving write may refuse
them: that is appropriate for a read-path guarantee and does not bypass the
production behavior being measured. The new consumer test normalizes omitted
empty slices through JSON, so parity measures the wire contract rather than a
Go nil-versus-empty distinction. It compares actual CLI entrypoints and MCP
responses with independently asserted domain identity and approval expectations.

## Judgment conclusions

1. **assumption-integrity — passed; approved in the native admission.**
   Coherent claim fields do not establish proof; a trusted signature establishes
   who/which execution the issuer asserted, not whether review criteria passed,
   whether the issuer told the truth or whether cognition was independent.
   Positive controls and one-variable corruption tests support those boundaries.
   B04 authenticates historical claim time under current pins; expired current
   operation validity does not erase a historically valid signed claim. The
   independent focused run includes `TestReviewTrustTimeHistorySurvivesExpiryWithoutHistoricalAuthorization`
   and revocation/tampering cases. Historical authorization remains
   `not-established`; live pins/grants still matter. This review's own identity
   and actor separation remain declared under repository policy, with no invented
   human confirmation or adoption.
2. **compatibility — passed; approved in the native admission.**
   The fix changes no CLI arguments, JSON schema, persisted record schema or MCP
   tool contract. It tightens the truth condition of existing verified fields.
   Declared records keep the separate signing gate and declaration semantics;
   valid authority with bad criteria retains identity but fails approval. Existing
   B04 historical time and current revocation semantics stay intact. Direct,
   rendered and HTTP MCP consumer tests verify the same assurance values.
3. **design-causality — passed; approved in the native admission.**
   The cause was disclosure checking claim fields without authenticating their
   signed envelope. D1 moves stored-proof and exact bundle binding into
   `verifiedAuthorityBlockers`, which both disclosure and gate evaluation use.
   This prevents one consumer accepting a damaged signature another rejects,
   while leaving semantic approval separate. Human confirmation can be verified
   only from the authenticated `VerifiedClaim`, so invalid proof cannot promote
   attribution indirectly. Each causal boundary has a negative regression and
   a valid positive control.
4. **documentation — passed; current operational wording reconciled.**
   Requirements, D1, historical failed validation and original start remain
   preserved. The qualification report distinguishes prior code, current tests,
   native closeout and final published CI/Security as separate evidence. The changelog states
   the user-visible correction without claiming cognitive independence. The implementer
   resolved my low documentation observation in the reviewed candidate: current
   Final Report/Residual risks now describe delivered B04, live pins/grants and
   unestablished historical authorization. Historical baseline prose is preserved.
5. **operability — passed; approved in the native admission.**
   Invalid proof produces bounded signature/bundle/authority blockers instead of
   verified wording. All human CLI consumers include the shared assurance
   rendering; JSON/MCP expose the same fields. B04 separately discloses historical
   claim validity, current operation validity, current trust and unestablished
   historical authorization. A user can distinguish a rejected proof from valid
   identity with rejected criteria. No new service, durable state, log exposure
   or deployment operation is introduced.
6. **scope — passed; approved in the native admission.**
   The original production change is the shared proof/bundle check plus retaining
   the declared signing gate. The current PR adds qualification, renames a test
   for the existing selector and updates delivery evidence/planning. No authority
   adoption, gate weakening, write-admission preflight, speculative migration or
   B02 work is bundled. B01 completion remains conditional on both specs being
   natively terminal; B02 starts only after merge.
7. **security — passed; approved in the native admission.**
   Signed bytes, Ed25519 algorithm/subject, trusted key pin and authority bindings
   precede verified disclosure. Revoked/untrusted pins and unsigned mutations
   fail; human roles require the existing live human-authority grant. Missing
   envelopes cannot use a legacy ledger exemption to obtain verified identity,
   because the authority checker still explicitly requires an envelope. Fixture
   keys are generated synthetic data, never retained credentials. No dependency,
   shell execution or new external authority surface is added. Final repository
   Security/dependency evidence remains a required publication condition.
8. **solution-proportionality — passed; approved in the native admission.**
   The smallest sufficient fix is reusing the stored-envelope verifier at the
   shared authority boundary. Duplicating cryptography in the renderer would
   invite drift; validating all criteria in the proof checker would erase valid
   identity on semantic rejection. The chosen implementation adds neither a
   policy version nor a new proof representation. A future independently signed
   identity object would require reconsidering this coupling; none exists in the
   inspected contract. Reverting the scoped production change is the documented
   rollback, without rewriting stored records.

## Anti-mechanization and findings

The cheapest false compliance is preserving coherent authority fields and a
human-looking prefix while damaging/removing the signed proof, then relying on
disclosure to call it verified. The focused proof tests and CLI/MCP consumer
test reject exactly that shortcut. The corpus README already documents related
`declared-independent-prefix`, `human-label-without-confirmation` and
`confirmation-reused-for-edited-content` cases. It does not register a dedicated
damaged-proof disclosure row; coverage is in the explicit domain/integration
tests above. This PR qualifies an existing gate rather than introducing a new
gate, and no missing behavioral enforcement is inferred from that catalog detail.

No critical, high or medium code/security findings were identified. The low
documentation observation about stale present-tense B04 residual wording was
resolved before admission. No accepted risk or
new deferred implementation obligation is created by this review.

`pose recurrence-check --tolerant --window-days 14` reports zero flagged/flapping
keys and one resolved cluster. No matching systemic recurrence escalation is
required. The assessed debt report has zero markers. Integration assessment
shows 63 unobserved consumers among 64 contracts; those broad static observations
are not treated as runtime failures or dismissed as proof of integration. The
actual affected MCP HTTP consumers have explicit exercised tests.

## Tool dispositions and closeout conditions

Required active tools passed for admission: native strict validate (global,
including the full affected component matrix), artifact-check, history-check,
knowledge-check and surface-check. Completion tools review-check and
closeout-check were deferred in the attestation and then executed successfully;
the guarded native transition produced a terminal closeout snapshot.

Recommended tools: suggest-review was executed independently; assess-design,
assess-discover, assess-integrate and assess-tech-debt were executed by the
implementer and their resulting observations inspected here. The structural
design and assessment outputs inform scope/risk; they do not replace behavioral
qualification or any judgment above. No recommended tool was silently used as an
automatic decision.

Decision: approved for the inspected subject and native closeout admission.
No unresolved finding or human adoption is inferred. CI and Security must pass
on the final published PR head, and native review/closeout verification must
confirm that publication before it is represented as ready for maintainer review.
The maintainer retains merge control.

## Approved native admission

Current native global strict execution on published commit
`ea53653b7f0aeebb8a22d394b9be3ce39eb8578f` passed 132/132 in 241.1 seconds,
with no failed, errored or skipped checks. The delivery-validation input names
that exact `git_head`; the sealed subject carries 129 current evidence entries,
including affected assurance integration/reachability and historical trust/reuse
integration. Current design observation inspected all 8 attributed files, with
zero material structural deltas. Artifact and surface strict checks have zero
error findings. Native check strict passed with 30 visible warnings; no rule or
policy was weakened to remove them. The attributed change set spans unrelated
commits; its attributed paths define this subject, not its whole base..head range.

- Admission bundle: `rvb-3bdcc9c37e034dc4`, digest
  `sha256:3bdcc9c37e034dc488b54bbe50902b2f730015d716904cdb007d2e8a13e43a0b`.
- Admission plan: `sha256:10eb56a14b9c3261d4a89b17b8af4dd109069f9c9ec0ecad69c20e5428646cda`.
- Independent attestation: `rva-3a33dadbf95db8b8`, recorded by
  `agent:independent-codex-pr182`. Native auto-attest left eight judgments pending;
  the reviewer explicitly concluded all eight and recorded all fifteen criteria.
- Native review verify and review-check: fresh and approved, no blockers;
  initial state `ready-to-close`. Guarded `pose close --apply` then completed.
- Native admission snapshot `b01-disclosure-closeout-admission.json`:
  `terminal: true`, `lifecycle_done: true`, `blockers: []`; current native review
  state `closed`. Spec lifecycle was written by the native transition as `done`,
  completed on 2026-10-15.
- Attribution explicitly identifies the independent agent as preparer, concluder
  and applier. Identity and separation remain declared, verified separation is
  `not-verified`, cognitive independence is `not-observable`, and confirmation is
  `none`. No human confirmation or maintainer adoption was fabricated.

Candidate CI run `38815135675` and Security run `38815135587` passed on that
published admission candidate. These do not establish checks on a later final
publication. After the implementer publishes lifecycle/matrix/roadmap summaries,
reconcile any superseding bundle using the unchanged implementation and exact
native evidence provenance, record the independent reaffirmation if needed, and
consume `b01-disclosure-review-verification.json` and
`b01-disclosure-closeout-check.json` for the latest published-state result.
B02 remains unstarted until this PR merges.
