# B29 replacement qualification — 2026-10-23

Native lifecycle: done, completed_at2026-10-23. Actual separate R6 source review and fresh reconciled-governance review approved; current native attestation and terminal verification passed, preserving the locked close transition. Final PR CI and merge remain pending.

Source checkpoint `7d9c07c` combines the operation-scoped update replacement with reviewed B30, B28 and B38. The original copier history and rejected delivery statements remain explicitly historical. Actual implementation and composition tests, failures, fixes and limitations are recorded in `2026-10-22-b29-update-preview-implementation.md`.

Fresh canonical native collection: 146/146 checks pass, zero errors, 208.0 seconds; command `pose validate --strict --report --json-out .pose/results/delivery-validation.json`, log `/tmp/pose-b29-composed-validation.log`. This is the registered matrix collection, not a substitution with direct Go tests. Direct full Go, vet, full affected-package race tests, six platform builds and compiled-CLI inotify mutation audit also pass. The audit observes filesystem events and does not claim syscall tracing or runtime testing on macOS/Windows.

Discovery refreshed pose-mcp at the composed source/matrix. Integration assessment retains heuristic unobserved-consumer findings and is not a clean contract certificate. Technical-debt assessment is INCOMPLETE because two legacy specs are invalid: `2026-08-15-pose-instance-engine-version-tracking.md` and `2026-08-15-pose-knowledge-durable-reference-type.md`; scanned content contains one covered TODO and zero uncovered markers, which does not prove unread sources are debt-free.

Actual separate source review was first attempted with Claude Opus, run `/home/go/.local/state/secondary-agent/20261023T040419-claude-3547062`; organization-disabled subscription access prevented execution. A separate same-vendor Codex source review was launched before the user's subsequent Claude-only instruction, run `/home/go/.local/state/secondary-agent/20261023T040719-codex-3642496`. No cross-vendor, human or cryptographic independence is inferred. Subsequent secondary reviews must use Claude Opus, per the user's current instruction.

The actual source review requested changes for three medium filesystem fidelity gaps. Regressions reproduced each before fixes; the implementation report records correction, full-suite/race/vet/build/audit evidence and conservative INCOMPLETE behavior where access, umask or hard-link effects cannot be safely evaluated. A further parent regression reproduced and corrected trailing-separator directory requirements. Source review remains a historical rejected attempt, not retroactively approved.

The user subsequently explicitly authorized resuming separate Codex secondary reviews because Claude configuration cannot be resolved now. This supersedes the interim Claude-only preference; preserve actual same-vendor execution attribution.

Pending: fresh native collection on corrected source, immutable artifact reconciliation, delivery surface gate, sealed final adjudication, native review verification and closure, dedicated PR and required CI. No release/tag publication, Harne8 adoption or shared-checkout/gitlink changes are authorized or performed by this qualification.


R2 source review confirmed original findings fixed and requested two additional conservative-uncertainty corrections. Both reproduced and fixed with operation and real shared-update tests; see implementation report. The subsequent146-check collection on e7b768e was145/146 due externally rate-limited release listing, not a passing final run. Fresh final collection follows the R2 fix commit, using configured authenticated release access.


Final corrected native collection passes146/146, zero skips/errors,206.4s; git_head3f03b36a0bdcb2b457cc899712c3952642bc0550, scoped provenance sha256:99d48e214fccd1ab4997cb0f15328aa40ae7938e48b5803c180c2079bb5233f3. Log `/tmp/pose-b29-final-validation.log`. Authenticated journey passed without test changes or waiver. Final affected update/operation race selectors pass CLI22.027s/model2.161s, `/tmp/pose-b29-final-race.log`. B38 main mergec9116ff postmerge CI39916673062 and Security39916673022 both succeeded.


Post-R3 fresh strict native collection:146/146, zero skips/errors,222.1s, git_headaebe238d6b73fd41cf8f3fc269032e8d092fef04, scopesha256:f7d3dad99efb7600f9ef5a5b8cc3e03e709012f9f883ca9f26efb95308d57bd7. Log `/tmp/pose-b29-final-validation.log` now records this last execution; original rejected-source outcomes are retained above. Affected full update/operation race selectors pass CLI20.867s/model2.164s, `/tmp/pose-b29-aebe238-race.log`. Native immutable report rangefe765cf..aebe238 records merge-composed paths. Required trace heading corrected for the parser; no requirement or source changed by this editorial correction.


Current final source24e3328d851f9714459ad27eee05c2f248304aa0 passes146/146, zero skips/errors,228.4s; scoped provenancesha256:f7ea85e418a612d5595ae06991d88d453c2214c024f8c56725ad549f5b2aa465. Final update/operation race passes CLI20.861s/model3.155s. All six CGO0 builds succeeded on this source (linux/darwin/windows,amd64/arm64), outputs `/var/tmp/pose-b07-validation-0bf729f/b29-24e3328-*`. The native collection includes full Go test/vet/build and all registered integration checks; actual inotify audit remains filesystem-event evidence, not syscall evidence. New provider/defaultACL/special-read guards and exact before/after evidence are recorded in implementation report and ADR.


Post-R4 reporting-fix source3570008dec67597a74e46079eb4d9e262255a596 passes146/146, zero skips/errors,223.8s; scoped provenancesha256:5b4e91d7b6e70a9ad958ff5f45b8a6685a2175b1a8960e300b6949b638892130. Fresh current operation/update race passes CLI24.342s/model4.202s, `/tmp/pose-b29-post-r4-race.log`. FIFO operation and real shared-update subprocess tests terminate and preserve the original FIFO, with no IN_OPEN event in the operation comparison watch. Earlier0error artifact/surface runs belong to24e3328; current gates and separate sealed adjudication remain pending.


Post-R5 recursive-removal sourcea7af88ff481088ba63f7a8aeeed572f55e6715a2 passes146/146, zero skips/errors,221.9s; scopesha256:4eaca93a30f971ff4e84605039f1da80541eb2f5e1bf84bf1812b09d656282c5. Current operation/update race passes CLI22.838s/model4.195s, `/tmp/pose-b29-a7af88f-race.log`. Prior native partial-deletion controls and real force-update retention failure/fix are recorded in implementation report. Current index, artifact/surface and actual sealed review remain pending.


Aligned immutable source rangefe765cf..a7af88f is recorded by the native report with full resolved SHA, preserving the historical aebe range record. Recollection after metadata alignment passes146/146, zero skips/errors,169.0s, unchanged git_heada7af88ff481088ba63f7a8aeeed572f55e6715a2, scopesha256:f4478c958054c41f9eeb57858b11b6f457028c50e97bdc821f910f0a7da44c6e. This removes the historical subject-head ambiguity without changing code or weakening provenance.


Current aligned artifact gate:114claims,0errors,598repository-wide orphan warnings (not debt-free certification). Surface gate:0findings. Current govulncheck reports no vulnerabilities, `/tmp/pose-b29-current-govulncheck.log`. Sealed rvb-c59b0c2cd1a61af8 digestsha256:c59b0c2cd1a61af8374f50cab417628cc27abf076f733436da242b92f1a45558, plansha256:95afc73c3aaf27978723e46a762977b1a5856386bf7ac6c0abf09233e80c73d3,143entries=142collected+structure, sourcea7af88ff481088ba63f7a8aeeed572f55e6715a2. Mechanical auto-attest preview succeeded without applying any judgment. Actual final separate adjudication is pending; no approval is inferred.


Actual resumed R6 session01a168c9-5c9b-76f1-9449-ce5c3ab3544c approved exactsealed bundlec59b0c2cd1a61af8, all15criteria and SD-cb046cda→D2 causality. Response is preserved verbatim. No new finding or accepted-low risk was introduced; historical rejections remain intact. This is actual separate same-vendor Codex execution, not authenticated/cognitive/cross-vendor or human independence. Approval requires current final PR CIgreen before merge. Native attestation/verification/transition remain the primary's next steps.

The recoverable all-in-one plan's startup-cleanliness condition was inspected after manual evidence preparation and reports the intentionally uncommitted governance/results. Continue the explicitly documented manual gated workflow: exact sealed review, review verify/check, delivery/artifact/trace gates and the native locked close transition, then one final metadata commit. No skip-evidence flag, manual status edit or weakened policy is used. All110 sealed entries were verified against actual current bytes by the reviewer.


Native attestation rva-d60fa5c05b92213f records the actual15passed judgments and SD-cb046cda→D2 mapping. Review verify reports ready-to-close/fresh/approved, review-checkpasses, and native `pose close spec:pose-update-dry-run-reports-the-whole-update` reports Closed with completed_at2026-10-23. No manual lifecycle edit was used. Postclose discovery/index and final terminal verification follow before metadata commit and PR publication; current PRCIgreen remains mandatory before merge.


Post-close lint found that the independent amendment ledger acknowledged only D2, although the original immutable 2026-10-10 start receipt already froze R1-R3 and D1. Preserve that receipt and the old D2 event exactly. Native amend appended a current-date baseline reconciliation authored @agent-codex, explicitly not retroactive execution precedence. A native transition then acknowledged historical D1 as withdrawn in favor of D2 and recorded its retrospective R1/R2 basis while preserving its original text/date. No requirements or implementation changed. Strict lint now passes with3amendment events,0failures and complete3-requirement trace. The existing closed review remains retained by policy, but its delta correctly marks decisions changed; obtain a fresh sealed governance adjudication rather than relying on retention.


Fresh separate governance session01a169a1-f9ba-74c2-b293-c25b0853559c approved newsealed rvb-08bd807629520cd3, all15actualcriteria and SD-cb046cda→D2. It verified unchanged110entries/implementation digest, original start bytes and initial D2 event preservation, current-date declared reconciliation and D1 withdrawal, valid node hashes and strict lint. No new finding or accepted-low risk. Current full source and canonical146/146 evidence unchanged; previous closed retention is explicitly not the basis of this new approval. Native fresh attestation and terminal verification follow.


Current reconciled native attestation rva-5f09aa2ff09cdaf2 binds actual15judgments and SD-cb046cda→D2 on rvb-08bd807629520cd3. Reviewverify closed/fresh/approved, review-checkpass, closeout-check lifecycle_done=true review.approved=true terminal=true next_action=none. Strict lint passes with3amendment events/0failures. Native completion date2026-10-23 preserved. Code remains unchanged from actualR6 source approval and current canonical146/146. Final metadata commit and external PRCI/merge remain pending, with no release/adoption/gitlink change.
