# Independent review, round 12: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `971288c7003f4cb6e657388285176a6dbe56994f`. The revised R3 mechanical gate passes and F7 is corrected. Exhaustive review found two further medium findings, **F8 and F9**, in the existing measurement consumer and its newly pinned cleanup trap. No implementation correction, seal, attestation, closeout, commit or push was made.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, maintainer authorized. This is a Codex/OpenAI execution reviewing Claude/Anthropic implementation, a different vendor. The `agent:independent-` prefix is an engine-required declaration until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied, not independently authenticated.

Reviewed the full branch diff, current correction and spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. The preceding review remains in commit 971288c7; these owned artifacts now report round 12.

The maintainer explicitly authorized reducing R3. This review evaluates plain helper-form omission/order and full-file hash pins, not arbitrary-shell cleanup analysis. Old counterexamples that defeat the helper using unusual syntax are now declared review-dependent limits and are not reopened as scanner defects. Re-pinning requires an actual reviewer judgment; a checksum does not authenticate review authorship. All ten pinned scripts were read, not merely hash-checked. That human pass exposed F9.

## Complete findings

### F8 — Medium: the measurement producer no longer delivers the quickstart's sourced dependency

Locations: `tests/quickstart/first-governed-loop.sh:18`, consumer `tests/quickstart/measure-clean-environment.sh:41`. Affected contract: **R2 integration** and the Intent constraint preserving what scripts execute/assert.

The existing clean-environment producer copies only first-governed-loop.sh to /measure-loop.sh, then invokes it with --binary from inside a clean container. The new unconditional source resolves ../lib/tempdir.sh; no helper is copied into that layout. The script therefore stops before the binary branch or any documented assertions. On the actual root-level container path, that dependency resolves to /lib/tempdir.sh, which the producer does not provide.

Reproduced the exact script-only copy layout outside the repository, without Docker or network. The current script exits **1** with a missing ../lib/tempdir.sh diagnostic. The origin/main version, copied the same way, executed with the same absolute branch CLI and working directory, exits **0** and performs its assertions. Both leave the supplied TMPDIR empty in these --binary runs. The absence of leaked files on the failed current copy does not preserve its behavior.

Expected action: deliver the helper at the script's expected location through this producer, or revise its dependency packaging/invocation. Exercise the existing measurement command path and re-review/re-pin the changed producer. Do not remove its assertions or treat the early failure as valid cleanup evidence.

Evidence: measurement-probes.py/json, script-only-current.log and script-only-original.log under the external evidence directory. Consumer search found this script-copy producer; ordinary direct CI/local invocations retain the helper and pass.

### F9 — Medium: a Docker cleanup failure skips host temporary-directory removal

Locations: `tests/quickstart/measure-clean-environment.sh:9`, newly adopted pin at `tests/hygiene/temp-cleanup.pins:7`. Affected contracts: **R1 failure-path cleanup**, and the human review premise of the own-trap exception in **R3**.

The EXIT trap runs docker rm before rm -rf "$work", with set -e active. If docker rm fails, Bash aborts the trap and never removes the host directory. This is a pre-existing trap defect exposed by reviewing the newly added whole-file pin, rather than another shell-parser counterexample. A matching hash freezes this failure path as well as its happy path.

Executed the **unchanged full measurement script** with a local Docker fault injector. Pull/image/create/copy return controlled successful responses, start returns 7, and rm returns 9. The script exits **9**, and its own TMPDIR contains the allocated tmp.* root. A paired control changes only rm to success: the same start failure produces exit **7** and an empty TMPDIR. Command logs prove container identity was assigned and rm was reached in both runs. No Docker daemon, real container or network was contacted, and no measurement report was fabricated.

Expected action: ensure host filesystem cleanup runs even if container removal fails, preserving appropriate error reporting; cover that path and re-review/re-pin the corrected file. This finding does not demand static analysis of arbitrary shell: it is the concrete manual allocation/trap review the new pin design delegates to the reviewer.

## Revised R3 and previous findings

R3's mechanical contract passes the independent complete-gate controls:

- A newly tracked intent-to-add script is rejected without the helper, with init after mktemp, commented or indented init, a one-line conditional init, missing source, or source after init. The valid plain source/init form is accepted.
- Appending a comment to each of the ten pinned files makes the gate fail, proving the whole file is covered.
- Removing a pinned script, removing its mktemp mention, or adding a nonexistent pin makes the gate fail.
- A changed pinned file fails before --repin; the explicit --repin command updates its actual whole-file digest, after which the gate passes. Every test mutation and re-pin was restored.

All ten committed hashes match. No current tracked *.sh lies outside tests/, scripts/, examples/ or install.sh; Git pathspecs include descendants. Helper and hygiene remain static exemptions with dynamic checks. The rule deliberately reads lines rather than shell and relies on review of new scripts; that limit is now accurately documented. F2 is retired under the maintainer-authorized requirement change, not claimed solved as an arbitrary-shell proof. F9 prevents approving the human review of one current pin.

F7 is corrected: the helper and all three real scripts return 0 and leave relative TMPDIR empty, both with CDPATH unset and exported as dot (eight runs). CDPATH no longer contaminates the capture. The shipped old-path control remains able to fail. F1/F3/F4/F5/F6 remain resolved in their original reviewed scope: twelve independent nested permission/failure scenarios clean up; the exact shipped setup reaches its markers and exits 0/1/1/1; INT/TERM yield 130/143 and clean up; old permissions and missing INT-handler mutations fail; the gate's own signal/failure paths clean up; and composition docs name pose_test_tmp_remove.

R1 holds for the helper's exercised matrix, but the newly pinned measurement trap fails it in F9. R2's ordinary direct runs pass; the script-copy consumer regression is F8. Restoring the three origin/main scripts makes the gate fail; the old quickstart without --binary returns 0 and leaves a directory, failing the empty-parent condition. The separate old-copy --binary positive control has no binary-building temporary directory and removes its normal work/keys, so it leaves TMPDIR empty. Those are different, intentionally recorded command paths.

D1 remains proportionate: a single owned root captures temporary work of Go, curl and git. Its new source dependency must also travel through existing consumers. SIGKILL independently exits -9 and leaves a root, an unavoidable trap limit requiring disclosure before closeout; Final Report remains Pending. The declared static limits are accepted scope boundaries; F8/F9 are observed defects, not accepted residual risks.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over CI globs, hygiene and TestLocalVerifyCoversCurrentCIGates pass. Quickstart, migration, installation, demo and quickstart --binary pass with absolute caller TMPDIR and leave it empty. Install also passes --binary/--previous 6.2.0 from another directory. Network access worked anonymously; no token was required or printed. Relative TMPDIR/CDPATH runs pass as described above.

CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene gate. Artifact-check exits 0: 13 claims are observed among 15 paths, with the two review artifacts as extras and no finding naming this spec. Implementation attribution is present. Tech-debt finds zero markers/uncovered items; recurrence flags zero keys. No API/MCP/REST/event contract changed, so assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full outputs, copies and executable probes are outside the repository in /tmp/pose-temp-independent-r12. The manifest contains exit codes, facts for this spec, every finding and SHA-256 hashes; it embeds no full repository-wide output.

Baseline real scripts, Go suite and gates completed before tracked mutations. Mutations were sequential and restored byte-for-byte, including the temporary intent-to-add entry and pins. All intentional leaks were observed then removed. Native assessment/index effects were retained externally and restored. Only the two reviewer-owned files remain modified; no code correction or persistent re-pin was made.

## Native closeout disposition

Prepared preview: `rvb-9127025bf1318349`, unsealed. No attestation. Strict delivery validation, review verify, review-check, close, lint-spec --strict and check --strict were not run because the medium findings prevent entering the conditional approval closeout. Lifecycle, tasks, trace, implementation plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- POSE bugfix review workflow and pose-review/pose-spec-closeout skills: controlled scope, effective gate value, contract compatibility and explicit dispositions.
- Security: temporary paths, permissions, signals, consumer dependency layout, pin integrity and evidence quality.
- Backend Go: branch build, vet and full suite, without Go-source edits.
- Documentation style and delivery evidence: honor the authorized R3 change and distinguish mechanical gate success from human own-trap review.
- Knowledge governance: consulted; no accepted residual-risk handoff.
- Frontend and inter-service contracts: not applicable. Surface/completion tools remain deferred because approval is blocked.

Resolve F8 and F9 before sealing or closing. No production or test correction was made by the reviewer.
