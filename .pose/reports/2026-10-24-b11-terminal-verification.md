# B11 — terminal closeout verification

Canonical spec `pose-closeout-terminal-verification`; baseline `94b4f4b9ddd138d4faf83ad48eab4fba04234414`, branch `impl/pose-b11-terminal-verification`. The amendment prerequisite is done and B08 remains preserved. No remote publication, parent gitlink update, release or policy adoption.

## Reproduction and correction

The before-fix CLI test TestTerminalVerificationDoneWithoutReviewIsBlockedAndReadonly failed: done without required review returned already-satisfied and a fabricated terminal plan. Local log `/tmp/pose-b11-repro.log` is supplementary, not delivery authority.

Done plans now use GetCloseoutState, including applicable historical review contracts, adoption cutoffs, opted-in amendment history, actions and configured federation. Lifecycle and terminal verdict are separate. Historical pipeline steps read not-run; the current verification query carries basis and limitations. Apply/resume and plain close on done are readonly, with no checkpoint/receipt/review/lifecycle repair. B12 already-satisfied still has empty steps_run/effects. Invalid history also blocks in-progress planning before sealing through the one prerequisite assessment. Transition effects are confirmed by callback at the actual successful publication, not exit 0 from a possible no-op; the renderer reports closed only with confirmed spec-closed.

## Tests and separate review

Domain/CLI/MCP tests cover required-review gaps, valid retained review, legacy cutoff, policy off, malformed policy/history, amendment gate before sealing, no-op preserving bytes and malformed old checkpoint, and false transition-effect prevention. Existing historical retention, B12 and amendment controls passed. Full initial suite identified two contract integration adjustments: catalogue the new diagnostic and preserve a link to repeat closeout-check; both corrected before final collection. No runtime Git engine changes: test commands use TMPDIR=/var/tmp to avoid the existing invalid /tmp/.git ancestor.

Actual separate execution `/root/b11_review` approved the technical delta, ran targeted domain/CLI/MCP/historical controls and a federated unknown-project probe. It also reviewed the transition callback and ran publication/operation/diagnostic controls. No blocking finding remains. Native declared identity `agent:independent-codex-b11-review` identifies this real separate agent execution; no human confirmation, cryptographic identity or cognitive independence is claimed. Full suite/scanners are executed by the primary, not attributed to the reviewer. Native attestation follows canonical results, fresh index and strict surface.

## Rules and boundaries

Applied backend-go (shared domain, error propagation), security (no authority/policy changes), documentation-style (EN/PT parity) and delivery evidence (registered integration, immutable attribution, current provenance). Consume knowledge:adr-sealed-review-bundles-review and knowledge:pose-b07-implementation. New not-run state is a breaking correctness fix; no new retrospective trace/delivery enforcement on legacy done scopes. No requirement, baseline, start or amendment history is rewritten. The generic adversarial corpus lacks a raw-done-terminal case, kept as an explicit low-priority follow-up; permanent regressions are registered.

Integration assessment retains 64 contracts, 1 observed consumer and 63 gaps. Debt assessment reports one covered marker, zero uncovered, but is incomplete due to two existing invalid specs. These inventory limits do not replace executed composition tests. No upstream contribution staged without consent.

Full `TMPDIR=/var/tmp go test ./...`: PASS after the catalogue/link/effect adjustments. `govulncheck ./...`: PASS, no vulnerabilities. Strict spec lint: PASS. Registered global validation, immutable artifact/surface gates and native review remain the authority for closeout.

## Canonical evidence and native closeout

- Full strict validation passed 152/152 (0 failed/errored/skipped), 262.1s, at source HEAD `2b8ab47d07326c30efb4559f94a9b69a8059353e`. The canonical result is `.pose/results/delivery-validation.json`, including `closeout-terminal-verification-integration` and full Go/build/vet producers.
- Artifact-check strict exit 0: claims=24, observed=24, no error findings; unrelated historical orphan warnings remain. Check strict passed with 30 warnings.
- Targeted race `TerminalVerification|ACloseRevalidates|ACloseNever` passed domain/CLI (1.139s/1.783s). Govulncheck reported no vulnerabilities and pinned CI gitleaks found no leaks in the source commit. No dependency or credential was added.
- After refreshing the index, strict surface reported targets=1, results=152, findings=0, provenance `sha256:083c0482347dedab0f56308fa296ce2fc411e950ebdcc41d33ba18a2e6c29780`.
- Sealed bundle `rvb-ec9e495960ba21da`, digest `sha256:ec9e495960ba21daf1ee4052e94f18696ac30d649c4141d76ddfc084479617cd`, contains 149 evidence entries. The actual separate reviewer applied approved attestation `rva-3acc8378f0515c97`, 15 criteria and mapping SD-526f0095 to D1, with explicit declared prepared/concluded/applied agent attribution. No human confirmation is claimed.
- Native review verify was ready-to-close/fresh/approved and review-check passed. `pose close spec:pose-closeout-terminal-verification` succeeded, writing done/completed_at `2026-10-25` through the guarded transition. No lifecycle or policy bypass.

Final assessment refresh, strict spec lint and terminal closeout verification are performed before the final evidence commit. Operational report/task updates do not rewrite the requirements, ADR decision, baseline, start record or amendment history. Parent Harne8 gitlink and remote publication remain untouched.
