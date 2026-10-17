# Independent review, round 3: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `4512073c97e076f07760de5e77482ff4f814ae83`. Exhaustive review completed before this decision: two medium findings remain. No persistent implementation fix, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic implementation, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. The maintainer-supplied implementer identity and session are not independently authenticated; the prefix does not establish signed execution provenance.

Re-read the current spec, full branch diff, correction diff, review/closeout skills, operating workflow and applicable rules. Built the branch CLI with the requested Go command and used that binary throughout. Requirements and Technical Plan were not edited by this reviewer. Previous-round records remain in commit 4512073c; these two owned review artifacts now describe round 3.

## Complete findings

### F2 — Medium: the scanner still accepts scripts that leave temporary roots

Location: `tests/hygiene/temp-cleanup.sh:42`, also lines 30, 35–39 and 44–50. Contract: **R3**. Previous finding remains open despite improved coverage of the seven named shapes.

A minimal ordinary regression is accepted:

```bash
a="$(mktemp -d)"
b="$(mktemp -d)"
trap 'rm -rf "$a"' EXIT
```

The scanner uses `any(...)` over assigned variables, so removing one root authorizes a script that leaks another. Independently executing the fixture returns 0 and leaves one tmp.* directory. Temporarily substituting this fixture into the already tracked migration-guides script also makes the **complete hygiene gate return 0**, confirming this is not an artifact of extracting its scanner.

The sweep executed twelve leaky fixtures, all accepted by the production scanner and all returning 0 with nonempty TMPDIR: partial cleanup of two roots, an extra unassigned mktemp, a variable containing a mktemp root plus `/file`, removal of only a child path, echo-only removal text, removal text inside a trap-body comment, an escaped literal variable, an overwritten trap, an unreachable trap, no-op helper initialization, unreachable helper initialization, and an overwritten helper EXIT trap. Correct trap and helper controls passed and left TMPDIR empty.

Expected action: establish coverage of every root and effective cleanup for accepted shapes. Reject unsupported syntax conservatively rather than accepting text as execution evidence. Add these negative cases. No adversarial case currently prevents these form-only passes; the seven built-in negative fixtures do not establish the stronger R3 claim.

### F5 — Medium: permission fixtures fail during setup and silently skip their named cases

Location: `tests/hygiene/temp-cleanup.sh:91`, with failure hidden at line 88. Contract: **R1 validation**, especially regression coverage of the F1 fix.

`files` executes `chmod 500 "$d"` **before** `mkdir "$d/a" "$d/b"`. As the non-root user used here and on ordinary CI runners, mkdir fails with Permission denied. Under `set -e`, every named success/assertion/error/unset-variable probe exits during setup. Neither 0400/0000 directories nor the intended final assertion, false command or unset-variable access are reached. The wrapper discards that exit code and accepts the emptied parent as success. Signal probes continue past failed setup because they do not enable set-e, so their restricted entries are absent too.

An xtrace of the exact shipped body confirms exit 1 at mkdir. Reverting **only** the helper permission command from `chmod -R u+rwx` to the old defective `chmod -R u+w` still makes the **complete hygiene gate return 0**. This is a reproduced missing regression guard, even though the current helper itself is fixed.

Expected action: create and verify the whole fixture before restricting access, distinguish setup failure from the expected scenario exit, assert the expected exit status for each probe and prove the old permission implementation fails.

## Previous findings disposition

- F1: resolved in implementation. Independent fixtures created nested paths and files before chmod, covering 0500, 0400 and 0000 across success, assertion, set-e and unset-variable exits. All twelve left TMPDIR empty. The shipped regression test remains defective as F5.
- F2: partially corrected, still open as detailed above.
- F3: resolved. Direct restricted-fixture probes returned INT 130 and TERM 143 promptly and left nothing. Temporarily removing only the helper INT handler now makes the full gate fail with status -2 instead of 130, proving that signal coverage can reject a regression.
- F4: resolved. The composition instructions now correctly name `pose_test_tmp_remove`.

## Requirements, D1 and complete-scope checks

R1's current helper behavior passed the independent permission and termination matrix. Its committed fixture is not trustworthy evidence for the newly claimed coverage (F5). R2 passed: each real script returned 0 and left its own TMPDIR empty. Migration and install ran from another directory; quickstart also passed from another directory using an absolute --binary. The install journey additionally passed with explicit --binary and --previous 6.2.0 from another directory. The API worked anonymously; no token was needed.

R3 still fails the substantive acceptance claim (F2). Restoring the three scripts from origin/main makes the static gate fail; running the old quickstart returns 0 but leaves a tmp.* directory, so its nonempty-directory check rejects it. The existing no-helper control and seven negative/three positive built-in scan fixtures run successfully, but do not cover the remaining defects.

The source/init order remains before the first mktemp, and subsequent cd and argument handling are unchanged. The full branch diff changes no script assertion: quickstart's trailing successful-path rm is replaced by the EXIT mechanism. Exporting TMPDIR captures Go work directories and explicitly allocated curl/git fixtures; these scripts do not rely on retaining the caller's original TMPDIR value. All tracked *.sh currently fall within tests/, scripts/, examples/ or install.sh; the Git pathspecs cover descendants. No current tracked-shell omission falls outside that scope.

D1 remains a proportionate choice: one owned root captures descendant outputs without maintaining a list. SIGKILL independently left its root, as expected for trap-based cleanup; this boundary still needs an honest residual-risk statement before closeout. The spec Final Report remains Pending. Future scripts that only look like cleanup are an observed enforcement defect, not an accepted residual risk.

## Deterministic checks and evidence

The branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, the hygiene command and the explicit TestLocalVerifyCoversCurrentCIGates run all passed. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke that exercised command. These passing commands do not override F2/F5.

Native artifact-check exited 0: all 11 declared claims are observed; the 13 observed paths also include the two previous review artifacts committed by the implementer. No native finding names this spec. Every implementation commit carries the POSE-Spec trailer. Tech-debt assessment found zero markers/uncovered items and recurrence-check flagged zero keys. No public MCP/REST/Protobuf/Kafka contract changed; assess integrate is not applicable.

Compact machine evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). It contains facts, exit codes, all findings and SHA-256 digests; complete outputs and executable probe sources are outside the repository at /tmp/pose-temp-independent-r3. Temporary mutations were restored byte-for-byte. Deliberately leaked probe roots were observed and then removed. Native assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-292b02431072a9f0`, unsealed. No sealed bundle ID or attestation ID exists for this round. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run: the approval condition was not met. No task, requirement trace, implementation plan or roadmap was marked delivered. No commit was created or pushed, and no upstream feedback was staged or submitted.

## Rules applied during review

- Change type: bugfix with CI and validation integration.
- Workflow and skills: .pose/workflows/review.md, pose-review and pose-spec-closeout.
- Security: temporary paths, signals, token handling and evidence integrity.
- Documentation style: spec, trap-composition instructions and review report.
- Backend Go: selected by the review plan; build, vet and full tests exercised, with no Go source change.
- Delivery evidence: passing checks are kept distinct from semantic requirement satisfaction.
- Knowledge governance: consulted; no residual risk was accepted and no handoff artifact was created.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2 and F5, then request a fresh independent review against the changed subject. Keep the spec open and prove both full-gate negative controls fail before any seal or closeout.
