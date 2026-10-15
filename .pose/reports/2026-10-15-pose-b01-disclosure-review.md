# Independent review: B01 review disclosure qualification

Date: 2026-10-15. Reviewer: `agent:independent-codex-pr182`, a separate agent
execution from the implementing agent. Scope:
`spec:pose-review-disclosure-requires-proof`, PR #182. This report records
independent source and behavioral review. It is provisional until current native
strict validation passes and the reviewer records an attestation against the
resulting sealed bundle. A prepared bundle or this report alone is not approval.

## Subject and authority

The local context resolver selects `proj.pose-dist`, the existing in-progress
spec and its canonical authority. I inspected R1–R4, D1, the original start and
implementation commit `164c2913`, the current `review_bundle.go`, and qualification
commit `fa47ecc8`. The original cryptographic fix came from merged #158; this PR
adds CLI/MCP qualification and includes the existing human-confirmation regression
in the `ReviewAssurance` integration selector. B04 was delivered by #178;
action-answer was closed in #179 and its publication complement merged in #181.
The implementation matrix correctly keeps B01 partial until this sibling closes
and keeps B02 waiting for the current delivery and merge.

The effective plan selects `spec-closeout@1`, `backend-review@1`,
`engineering-judgment@1` and `high-criticality-review@1`: fifteen criteria,
including eight judgment criteria, critical band, different-actor independence.
The sealed bundle and plan digest will be taken from the current native run;
the preliminary prepared identity is deliberately not treated as final.

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

1. **assumption-integrity — passed on inspected behavior, pending native seal.**
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
2. **compatibility — passed on inspected behavior, pending native seal.**
   The fix changes no CLI arguments, JSON schema, persisted record schema or MCP
   tool contract. It tightens the truth condition of existing verified fields.
   Declared records keep the separate signing gate and declaration semantics;
   valid authority with bad criteria retains identity but fails approval. Existing
   B04 historical time and current revocation semantics stay intact. Direct,
   rendered and HTTP MCP consumer tests verify the same assurance values.
3. **design-causality — passed on inspected behavior, pending native seal.**
   The cause was disclosure checking claim fields without authenticating their
   signed envelope. D1 moves stored-proof and exact bundle binding into
   `verifiedAuthorityBlockers`, which both disclosure and gate evaluation use.
   This prevents one consumer accepting a damaged signature another rejects,
   while leaving semantic approval separate. Human confirmation can be verified
   only from the authenticated `VerifiedClaim`, so invalid proof cannot promote
   attribution indirectly. Each causal boundary has a negative regression and
   a valid positive control.
4. **documentation — passed subject to current operational refresh.**
   Requirements, D1, historical failed validation and original start remain
   preserved. The qualification report distinguishes prior code, current tests,
   pending native closeout and final published CI/Security. The changelog states
   the user-visible correction without claiming cognitive independence. I asked
   the implementer to refresh the current Final Report/Residual risks: B04 is
   delivered, so present-tense wording that historical trust/reuse still await
   B04 is stale. Preserve historical baseline prose; update current reporting.
5. **operability — passed on inspected behavior, pending native seal.**
   Invalid proof produces bounded signature/bundle/authority blockers instead of
   verified wording. All human CLI consumers include the shared assurance
   rendering; JSON/MCP expose the same fields. B04 separately discloses historical
   claim validity, current operation validity, current trust and unestablished
   historical authorization. A user can distinguish a rejected proof from valid
   identity with rejected criteria. No new service, durable state, log exposure
   or deployment operation is introduced.
6. **scope — passed on inspected behavior, pending native seal.**
   The original production change is the shared proof/bundle check plus retaining
   the declared signing gate. The current PR adds qualification, renames a test
   for the existing selector and updates delivery evidence/planning. No authority
   adoption, gate weakening, write-admission preflight, speculative migration or
   B02 work is bundled. B01 completion remains conditional on both specs being
   natively terminal; B02 starts only after merge.
7. **security — passed on inspected behavior, pending native seal.**
   Signed bytes, Ed25519 algorithm/subject, trusted key pin and authority bindings
   precede verified disclosure. Revoked/untrusted pins and unsigned mutations
   fail; human roles require the existing live human-authority grant. Missing
   envelopes cannot use a legacy ledger exemption to obtain verified identity,
   because the authority checker still explicitly requires an envelope. Fixture
   keys are generated synthetic data, never retained credentials. No dependency,
   shell execution or new external authority surface is added. Final repository
   Security/dependency evidence remains a required publication condition.
8. **solution-proportionality — passed on inspected behavior, pending native seal.**
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

No critical, high or medium code/security findings were identified. One low
documentation observation is the stale present-tense B04 residual wording; the
implementer was asked to reconcile it before final closeout. No accepted risk or
new deferred implementation obligation is created by this review.

`pose recurrence-check --tolerant --window-days 14` reports zero flagged/flapping
keys and one resolved cluster. No matching systemic recurrence escalation is
required. The assessed debt report has zero markers. Integration assessment
shows 63 unobserved consumers among 64 contracts; those broad static observations
are not treated as runtime failures or dismissed as proof of integration. The
actual affected MCP HTTP consumers have explicit exercised tests.

## Tool dispositions and closeout conditions

Required active tools are native strict validate (global and affected component),
artifact-check, history-check, knowledge-check and surface-check. Their current
results must pass in the final collected evidence before attestation. Completion
tools review-check and closeout-check remain deferred until the attestation.

Recommended tools: suggest-review was executed independently; assess-design,
assess-discover, assess-integrate and assess-tech-debt were executed by the
implementer and their resulting observations inspected here. The structural
design and assessment outputs inform scope/risk; they do not replace behavioral
qualification or any judgment above. No recommended tool was silently used as an
automatic decision.

Provisional decision: no implementation blocker, ready for current mechanical
evidence collection and independent attestation after the current documentation
refresh. Final approval, bundle/plan/attestation identities, native verify and
closeout result will be appended after the subject is stable. CI and Security
must pass on the final published PR head before the delivery is represented as
ready for maintainer review/merge. No merge is authorized by this report.
