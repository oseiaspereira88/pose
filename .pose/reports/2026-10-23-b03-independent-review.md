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