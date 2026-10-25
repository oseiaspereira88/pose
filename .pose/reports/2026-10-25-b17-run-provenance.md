# B17: Signed delegated run provenance

## Authority and scope

Canonical spec `pose-delegated-review-run-provenance`, baseline `3e4c26d` after the maintainer merged PR #205. Preserved R1–R4, reconciled accepted ADR and B01/B04/B16 prerequisites, incorporated prospective P1–P6 as R5–R10 before native start. Consumed knowledge:review-assurance-requires-cryptographic-proof and knowledge:adr-sealed-review-bundles-review.

## Implementation

Capture explicit executor-attested implementation/reviewer baselines before execution. Bind project/audience/worktree, sealed bundle/scope/kind/commit, complete adapter configuration and argv template, actual argv/executable digests, brief body/template/notes, planned stdin and actual accepted pipe-write bytes, raw and retained redacted output/transcript digests and completeness. Preserve logical argv0 and resolve relative executables in the sealed worktree. Session values and private keys are not published. The signed executor declaration is not a vendor receipt or evidence of cognitive independence.

Reuse the existing native issuer and current pin grant to sign separate immutable run-proof envelopes. Legacy records do not acquire invented provenance. Exact run/proof references are signed into attestations, and authority bindings derive from the captured run. One verifier rechecks signed sources, project/worktree/config/audience/current time, complete attempt/continuation receipts and current sealed inputs before admission or disclosure. Vendor OR model difference follows the accepted contract, with known normalized identities and explicit trust; same vendor/model and unknown baselines refuse. Human requirements cannot be satisfied by machine runs.

Signed audit records may describe interrupted or unstarted executions with complete_review false. Completed bytes with no unambiguous draft decision also remain incomplete review evidence. Negative/ambiguous drafts cannot support approvals; the attestation cannot predate its completed run or authenticated proof (with only timestamp representation precision). Complete criteria/tools/evidence/preflight still apply. CLI verify/review-check/run-proof and MCP/Attention disclose the same current assurance and safe source identifiers.

## Actual separate review and regressions

Actual separate execution `/root/b17_review` inspected source, spec, ADR and applicable rules and ran independent overlays. High findings independently reproduced and corrected: a negative draft could support approved attestation, a deleted initial attempt receipt retained machine assurance, and an attestation could predate the referenced run/proof. Final independent probes passed. A medium budget finding by inspection was corrected by observing executable/argv before subtracting setup from remaining resume wall-clock. Legacy LedgerVersion eligibility, audit state/complete flags and both output channels were also checked. A later relative-shebang compatibility finding was independently reproduced: using the resolved target as exec path changed script $0. Preserve the original selected exec path and logical argv0, resolving only the hash observation, with a permanent regression. No before-fix execution is claimed for inspection-only findings.

Initial independent targeted tests passed domain 2.619s/CLI 0.488s/MCP 0.232s; later final-source review and native attestation follow canonical sealing. Test fixtures initially omitted tracked repo-map in the checkout or staged policy changes after the callback's initial git add; corrected fixture setup without weakening product checks. CLI/MCP fixture attestations now supply actual synthetic criterion conclusions and all planned tool dispositions. No real vendor CLI, project key, adoption or external runner was invoked.

## Validation and contract extension

Focused domain/CLI/MCP tests passed, including signature/binding, normalized OR rule, prefix-only/human refusals, corruption, revocation/expiry, negative/ambiguous draft, lost receipt, causal time, actual partial stdin, unstarted/partial audits, current project/config, immutable publication and Attention parity. Full Go suite passed after documenting the intentional second live trust bar. Race passed domain 7.300s/CLI 2.342s/MCP 1.795s. Govulncheck found no vulnerabilities. Final canonical matrix, source artifact reconciliation and native gates remain pending.

The exact live-policy guard retains its historical test name for trace compatibility and now permits only RequireSignedAttestations and DelegatedReview, with a positive old-sealed-bundle opt-in regression. This prospectively extends the earlier single-bar invariant without editing done specs or rewriting historical records. Every other verdict field remains sealed. No broad allowlist waiver was introduced.

## Limits

Current trust is an authorized executor's signed declaration. Raw bytes are hashed by the executor; retained redacted artifacts are directly reverified. Pipe acceptance does not prove vendor consumption. Signing does not establish semantic approval, erase rejection history, reset quota, migrate legacy data or authorize a human role. Worktree binding is local to the origin; portable CI/Harne8 runner trust is not qualified here. B18 owns containment; worktrees and hashes are not a sandbox. Execution fixtures were qualified on Linux; no Windows vendor/runtime efficacy is claimed.

B19 owns catalog adoption, compatibility/cutoffs/rollback, full typed conclusion validation and application. This feature recognizes the opt-in proof bar in the implementing engine but does not adopt it in this instance or qualify older/mixed engine versions for adoption. Keep CLI/MCP builds compatible before enabling any future adoption. Independent debt assessment remains incomplete due to two preexisting invalid specs (one covered marker, zero uncovered); integration inventory remains 64 contracts/1 observed/63 gaps. No upstream contribution was staged, and the parent Harne8 gitlink, release and remote publication remain unchanged by this implementation.
