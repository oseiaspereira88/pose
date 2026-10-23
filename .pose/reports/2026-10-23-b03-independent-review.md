# B03 independent review history

## R1 prospective architectural advice

Actual separate same-vendor run 20261023T180459-codex-3011272. Advice preceded implementation; no sealed delivery approval inferred.

Recommend a **versioned prospective presence contract**, with explicit migration before new human answers under legacy `require_presence: true`. B03 is not ready for implementation until its ADR resolves scope, bindings, and compatibility.

This is a same-vendor Codex source review of `impl/pose-b03-proof-presence` at `c4f814b82bd39223699f82979568fc973abcfcd0`. I cannot independently attest the runtime model designation. No child agents, tests, native commands, writes, sealing, or attestation were performed. The two existing assessment-file modifications were left untouched. **This is architectural/security advice, not final approval.**

**Findings**

- **High — presence enforcement depends on the supplied emitter.** [ResolveActionRequest](/tmp/pose-b03-work/pose-mcp/internal/pose/action_resolution.go:195) invokes presence enforcement only through [verifyActionSignature](/tmp/pose-b03-work/pose-mcp/internal/pose/principal_keys.go:181). An issuer-only human answer can pass without presence. Under declared assurance, an unsigned human answer also avoids the check. Universal enforcement must therefore occur after proof verification and before append, independently of identity-assurance mode.
- **High — issuer authority proves no physical presence.** [ActionAuthorityClaim](/tmp/pose-b03-work/pose-mcp/internal/pose/action_request.go:386) contains no admitted presence proof. Pins, human grants, principal labels, execution labels, and Portal authentication cannot supply it. Native `issuer_key.go` signs review attestations; I found no production action-claim signer there. Native and external action claims must receive identical verifier treatment.
- **Medium — policy writers need explicit treatment.** [ActionPolicyDocument.Write](/tmp/pose-b03-work/pose-mcp/internal/pose/principal_keys.go:298) retains raw settings, writes selected typed fields, and validates the full policy **after publication**. Merely adding a typed scope field will not reliably implement migration. Validate rendered policy before publishing; preserve scope/version through identity add/remove and role edits.
- **Medium — current disclosure overstates the contract.** [Doctor](/tmp/pose-b03-work/pose-mcp/internal/cli/doctor.go:1011) can suppress missing-key warnings because a human issuer grant exists, yet elsewhere says presence-ineligible keys mean no answer is accepted. Neither accurately describes every current branch.

**Recommended decision and alternatives**

| Option | Tradeoff |
|---|---|
| **Recommended: action-policy schema 2 with explicit `universal` / `sshsig` scope** | Old binaries reject schema 2 instead of ignoring a new security field. Keep presence disabled by default; require scope when enabled. |
| Universal meaning imposed on schema 1 immediately | Closest to original R5 wording, but newly rejects issuer-only and declared answers. Requires an explicit compatibility decision; cannot be assumed patch-compatible. |
| Explicit SSHSIG scope after migration | Preserves existing channel behavior, but deliberately narrows the original universal-sounding commitment. Must disclose issuer presence as unproved. |

For schema 1, preserve historical interpretation and expose `legacy-sshsig-only`. For **new human answers with legacy presence enabled**, return a migration-required diagnostic until scope is explicitly selected. This avoids silently perpetuating the bypass or silently adopting universal policy.

Reconcile foundation R5/D3 as the broader stated requirement and schema 1 as the narrower delivered behavior. D2’s “two emitters of one proof” supports common identity assurance; it does **not** establish equivalent physical-presence evidence. Record that reconciliation prospectively without rewriting completed foundation evidence. Release classification remains an explicit ADR decision because stricter writes affect compatibility.

For universal scope:

- Accept a registered, cryptographically valid SK signature asserting user presence.
- Reject issuer-only presence as unsupported with today’s claim contract.
- A combined SK signature plus issuer claim is viable only when **both independently verify and bind the same operation**. Any supplied invalid proof must fail; no fallback to the other proof.
- Apply the human requirement to humans. An agent must neither inherit mandatory human touch nor receive human-presence assurance.

**Proof and historical boundaries**

[VerifySSHSignature](/tmp/pose-b03-work/pose-mcp/internal/pose/sshsig.go:253) includes SK flags in signed bytes and returns success only after verification. Preserve that ordering. `UserVerified` must not substitute for `UserPresent`.

The current SSH statement binds project, request/version, actor, answer, reason, and idempotency key, but has **no explicit audience**. Issuer claims bind audience but not idempotency. Before admitting a universal combined-proof path, define audience linkage and operation correspondence. Do not assume those bindings already exist. SK verification authenticates an asserted presence flag under the registered-key trust model; it does not independently attest hardware provenance or identify who touched it.

Preserve [verifyActionAnswers](/tmp/pose-b03-work/pose-mcp/internal/pose/action_request.go:555): historical proof verification, recorded-time precision, current issuer pins/grants, and SSH verification after key removal. Do not apply today’s presence policy retrospectively. Any new contract metadata used to claim historical compliance needs authenticated binding; an editable event label cannot establish it.

**Implementation boundaries and negative controls**

Keep policy parsing/validation and common presence admission in `action_resolution.go` / `principal_keys.go`; historical verification and disclosure in `action_request.go`. Update CLI resolve/identity/doctor/help, MCP handler and tool schema in `server.go`, golden catalog, documentation/locales/scaffold copies, and validation-matrix producers. MCP already carries signature and issuer proof together; its schema currently describes them as alternatives.

Required regression pairs:

- Human/agent × declared/verified × presence on/off × universal/SSHSIG.
- Plain SSH, untouched SK, touched SK, UV-only, edited flags, invalid signature, wrong registered actor.
- Native/external issuer-only, missing/partial envelope, invalid pin/grant, unsupported presence fields.
- Combined proofs with mismatched actor, project, audience, request/version, answer, reason, or operation; either proof invalid.
- Every refusal leaves journal bytes/revision and satisfaction unchanged.
- Migration, unknown versions/scopes, contradictory settings, policy-edit round trips, and invalid-policy rejection before publication.
- Historical declared/issuer/SSH records retain truthful assurance after policy changes; B01 tampering and B04 temporal regressions remain intact.
- CLI/MCP parity; previews disclose pending verification without claiming presence.

Existing synthetic SK tests exercise signature semantics without hardware. They provide useful fixtures, but do not prove real device touch.

Applied rules: POSE review workflow, security, Go backend, documentation style, and recurrence guidance. Assessments, validation, vulnerability/secret scans, bundle digests, and sealed review remain unexecuted under this read-only scope. No Harne8 adoption or release publication is implied.

## R2 source review — actual separate same-vendor execution

Run 20261023T193719-codex-3115813. Historical rejection preserved verbatim:

**CHANGES REQUESTED — one medium source finding.**

**Medium — schema-1 scope rejection misses case-insensitive aliases.** In [`decodeActionPolicy`](/tmp/pose-b03-work/pose-mcp/internal/pose/action_resolution.go:92), struct decoding accepts case-insensitive JSON names, but the subsequent map lookup checks only exact `presence_scope`.

Source-derived reproducer:

```json
{"schema_version":1,"require_presence":true,"PRESENCE_SCOPE":"universal"}
```

The decoder accepts this and populates `PresenceScope`, violating the explicit contract that schema 1 must refuse the scope field. Older engines also accept this schema-1 document while ignoring the new setting. The current migration guard still blocks new human answers; this finding does **not** establish a current admission bypass.

**Smallest fix:** detect scope-field presence using the decoder’s case-insensitive matching semantics, including empty/null values. Add regressions for uppercase/mixed-case spellings with presence enabled and disabled.

Otherwise, the reviewed source preserves the intended boundaries: every supplied proof verifies before central presence admission, issuer authority does not establish presence, agents remain outside human-touch requirements, invalid rendered policies are checked before publication, and historical observations preserve B01/B04 limitations.

Evidence chronology is qualified: the before-fix log reproduces the legacy bypasses and invalid-policy publication; the domain-regression failure concerns native-fixture directory permissions, corrected in current source. Later focused/composition and race logs pass. The earlier full-package run passes domain/MCP but fails CLI’s known-key fixture; current source corrects that fixture and the focused compatibility log passes. Full canonical validation remains pending.

Applied the POSE review workflow and security, Go backend, documentation, knowledge, and delivery-evidence rules. Reviewed `c4f814b..4dc8277`, including prospective `06ebca5`, new tests, foundation contracts, and supplied logs. Native assessments, bundle/sealing, attestation, and test execution were deferred under your read-only instruction. No writes or child agents; existing modifications were left untouched. This is a source review, without delivery or CI-clean claims.

## R3 current-source sealed adjudication — actual same-vendor execution

**Decision: APPROVED**, conditional on all required final-head PR CI checks being green before merge.

This is the actual judgment of this Codex/OpenAI review session. The requested target is `gpt-6.1-sol`; I cannot independently authenticate the runtime model designation or reviewer identity. This is same-vendor review, with no claim of human, cognitive, or cross-vendor independence. I performed read-only source and evidence inspection; no tests, native mutations, writes, child agents, attestation, or publication.

The reviewed identities are:

```text
Bundle: rvb-343f8deb0f9be466
Bundle digest: sha256:343f8deb0f9be4664b932a8e223a1b4fad226b594d9204e92a4117e7501cea29
Plan: sha256:9f366d6323a3fc63a615c75ab98ace7a96781e05deec7f47fd64935fcde75d48
Implementation: sha256:ddf8a7f55396b7004853d33bb981b994edbeb07669f760fe849ee51adf0cc406
Patch: sha256:1c08ebb307d469c763cce6b3189239d42cc7f92ecee5e6fd611bceddff7a2a22
Tree: sha256:4fd4b403ddda22e4f86b4529fe183d8de69d8a6e543f320be48c5912dde8d5fe
Source: ae54756f7e8f41f661e40326e2a05821502e88f7
Baseline: c4f814b82bd39223699f82979568fc973abcfcd0
Change set: cs-9e6b04a9e595
```

I independently recomputed those five digests. All 25 subject entries match their hashes and committed source bytes; consumed file hashes and all four semantic-section hashes match. The managed-start baseline matches the sealed R1–R8/D1–D2 basis. All five foundation specs remain byte-identical to baseline.

The 143 sealed canonical evidence records match the supplied structured results and source provenance; the additional design record binds the same implementation. Current strict validation passes **147/147**, with no skips or errors. The permanent three-package presence producer passes. Supplied final race and govulncheck logs pass. These are inspected collector results, not executions performed by this reviewer.

| id | disposition | individual reason |
|---|---|---|
| assumption-integrity | passed | Disclosure distinguishes registered-key assertions from hardware provenance, issuer authority, historical authorization, and physical touch. Synthetic fixtures are correctly qualified. |
| backend-concurrency | passed | Admission precedes append; existing locked revision, idempotency, and open-state checks remain intact. Refusal tests preserve journal bytes/state; supplied race results pass. |
| backend-contracts | passed | Schema 2 explicitly selects scope; enabled legacy human writes require migration. Older engines reject schema 2, and breaking/major classification is recorded. |
| backend-input-errors | passed | Shared decoding rejects invalid versions/scopes and folded schema-1 aliases. Proof failures, missing assertions, unsupported issuer presence, and migration have actionable diagnostics. |
| backend-integration-impact | passed | CLI and MCP reach shared domain admission. The registered producer exercises all three packages, and surface evidence binds the current capability and source. |
| backend-observability | passed | Returned diagnostics and presence observations expose scope and limitations without fabricating presence or recording a successful refusal. |
| compatibility | passed | Prospective ADR explicitly reconciles foundation R5/D2. Historical verification is preserved; migration chooses no operator policy implicitly. |
| correctness | passed | Inspected 112 admission combinations, mixed-proof binding controls, alias regressions, invalid-edit protection, historical tests, and current canonical results support the behavior. |
| design-causality | passed | Explicit versioning resolves ambiguous channel enforcement; centralized admission closes emitter-dependent bypasses. The complete structural mapping appears below. |
| documentation | passed | English/Portuguese root and scaffold presence documentation matches runtime semantics, migration, preview behavior, and assurance limitations. |
| operability | passed | Previews disclose `not-evaluated`; migration is checked before CLI signing. Identity and doctor output provide explicit remediation. |
| requirements | passed | R1–R8 have inspected relevant tests and current integration evidence; strict lint reports eight trace entries with none missing. |
| scope | passed | The four attributed commits reconcile all 29 declarations with observed changes. No foundation rewrite, Harne8/gitlink change, policy adoption, or release is included. |
| security | passed | Every supplied nonempty proof verifies before presence admission. Universal human admission requires registered SK user presence; issuer authority and UV alone cannot substitute. Folded aliases cannot override typed presence edits. |
| solution-proportionality | passed | A versioned policy, shared admission function, and derived observation address the defect without inventing an issuer presence protocol or historical receipt. Any future issuer presence mechanism requires a new authenticated contract. |

The sealed plan contains **one** material structural fact:

| SD-id | Rn or Dn | causal reason |
|---|---|---|
| SD-0f985602 | D2 | The added prospective ADR defines schema-2 scope, explicit legacy migration, proof ordering, binding boundaries, and historical limits before implementation, providing the decision basis for the changed presence contract. |

Tool dispositions below refer to inspected primary-collected evidence:

| Tool | Requirement | Disposition |
|---|---|---|
| artifact-check | required | Passed reconciliation: 29 claims match observations; 693 warnings remain disclosed, with no scoped attribution error. |
| surface-check | required | Passed: zero findings; current composed-capability evidence and production entrypoint inspected. |
| validate — global | required | Passed: strict 147/147 on `ae54756`. |
| validate — pose-mcp | required | Covered by the same global run’s module-wide test, build, vet, and registered integration results. |
| review-check | required, completion-only | Deferred until the primary records this actual response in a fresh attestation. |
| closeout-check | required, completion-only | Deferred until attestation and review verification complete. |
| assess-design | recommended | Inspected sealed derived evidence and its single material fact; implementation binding matches. |
| assess-discover | recommended | Inspected initial freshness reuse and current refresh after source/matrix changes. |
| assess-integrate | recommended | Inspected; heuristic reports 64 contracts and 63 unobserved consumers, not proven integration failures. |
| assess-tech-debt | recommended | Inspected; explicitly incomplete for two older invalid specs. No debt-free conclusion. |
| suggest-review | recommended | Native rerun skipped under read-only instructions; selected backend overlay, installed workflow, and applicable rules inspected directly. |

Applied rules: review workflow; security; Go backend; documentation style; delivery evidence; knowledge governance; base recurrence guidance. Frontend and infrastructure rules are inapplicable. Recurrence scanning was not rerun under the native-execution restriction; no current escalation-clear claim is made.

**No new findings or accepted residual findings.** R2’s medium alias finding is corrected, including Unicode-fold aliases and canonical typed policy writes. R1 advice and R2 rejection remain historical records, not approval evidence.

Assessment coverage limits, component-owner/unmapped-path warnings, synthetic SK limitations, and platform-build-versus-runtime limits remain disclosed. Completion-only review verification and closeout are still pending. This approval does not establish final-head CI completion, authorize policy adoption, or publish a release.