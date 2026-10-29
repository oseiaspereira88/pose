# Delegated review capability — implementation review round 2

Source: separate Codex execution `20261029T050403-codex-1122168`; same-vendor limitations remain explicit.

**Changes requested: one high-severity blocker remains** in `17eb6e25..24b75f3a`.

I’m GPT-6-based Codex; an exact serving-model identifier is not exposed to me. This separate execution remains same-vendor, with correlated-model limitations, and provides no Claude or cross-vendor assurance.

1. **High — Disabling delegation can bypass the protected baseline’s signed-run requirement.**
   [review_policy_baseline.go:245](../../pose-mcp/internal/pose/review_policy_baseline.go:245), [review_bundle.go:3184](../../pose-mcp/internal/pose/review_bundle.go:3184).

   **Trigger:** an adopted baseline uses declared identity assurance, and the governed change sets `delegated_review: false`. The correction restores `different-actor` in the plan, but attestation validation requires signed-run proof only when the **current** policy flag is true.

   **Consequence:** the disabling change can satisfy its restored floor with `agent:independent-anything` and no run reference, subject to the other ordinary checks. Requiring an attestation signature alone does not establish a signed delegated run. The baseline therefore preserves the independence enum while losing its adopted proof requirement.

   **Fix:** preserve the baseline’s run-proof obligation specifically for review of the disabling change, through shared admission and verification. Add an end-to-end regression rejecting a prefix-only approval after disabling delegation. Keep the maintainer-selected live B17 barrier for historical approvals; this finding requests no grandfathering.

The four round-one corrections are present:

- **Baseline floor:** restored, but incomplete end-to-end as described above.
- **MCP escalation:** typed preview reaches authorized apply and retains a sanitized request identifier.
- **Configuration race:** activation parses and validates the captured comparison bytes.
- **Routing:** incompatible candidates are skipped before selection, allowing later independent candidates.

The existing positive regression log and four mutation logs support those corrections; mutation failures are behavioral assertions, not compilation failures.

I found no additional blockers in the inspected adoption/consent, exact issuer binding, recovery, routing/wait, run/attempt/task boundaries, disclosure/digest compatibility, legacy-history, or distributed documentation paths. The canned-approval corpus case correctly guards disclosure: local executor signing authenticates a declaration and bound artifacts, not review cognition or a vendor receipt.

Applied the review workflow and Go, security, documentation, delivery-evidence, and knowledge-governance rules. Context resolved to `24b75f3a`. Assessment publishing, suites, sealing, and attestation were excluded by your read-only instructions.

**No final green validation or approval is claimed.** The observed [final race log](/tmp/pose-roadmaps-evidence/adoption-race-final.log) ends in a Git-snapshot failure; registered validation remains unverified here. I made no edits, Git writes, agent/vendor launches, issuer changes, or attestations.

## Author dispositions

- R2-F01: seal the disabling review's protected run-proof obligation, bind it into both plan digests and enforce it through shared admission, verification and assurance. Runtime routing, escalation and automatic conclusion honor that protected review without re-enabling unrelated off-capability work. The end-to-end negative/signed-prefix and positive automatic-run regressions pass.
- Additional correction: reuse an existing project-specific native issuer through LoadIssuerKey before exclusive creation; the native duplicate/missing diagnostics do not wrap filesystem sentinel errors. The accepted-init journey now opens a second fresh checkout and proves key bytes are preserved.
- Additional correction: unavailable-route escalation identity no longer depends on a temporary request's requested_at. The regression crosses a timestamp boundary explicitly and MCP retry opens exactly one request.
- The non-Git domain sealing fixture now creates a real repository, avoiding transient ancestor Git markers while retaining the fail-closed production behavior.
- Four isolated round-two mutations fail their behavioral regressions without compile errors: protected proof, protected automatic conclusion, issuer reuse and stable escalation.

Registered final validation and a new sealed review remain required; this report is not an approval.
