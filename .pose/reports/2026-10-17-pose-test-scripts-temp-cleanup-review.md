# Independent review, round 6: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `f990ccdd331b34706dc85bbff7043580066f4ca3`. Exhaustive sweep completed before deciding: F2 remains a medium finding, with eleven accepted executed leaks including a current tracked script. No separate new finding was identified. No persistent implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied and not independently authenticated; the prefix does not establish signed execution provenance.

Reviewed the full current branch diff, correction diff, spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. Previous-round records remain in commit f990ccdd; these owned files now report round 6.

## Complete findings

### F2 — Medium: accepted cleanup forms still leave files on ordinary execution paths

Location: `tests/hygiene/temp-cleanup.sh:212`, with related checks at lines 43–95, 144–157, 160, 197–199 and 228. Concrete existing-script location: `examples/demo/record.sh:34`, `:36`, `:46`. Affected requirement: **R3**, with an observed R1 failure in the existing demo script.

The current scanner rejects all 39 external leaking fixtures retained from rounds 3–5. Its 57 negative fixtures, 5 positives and rebound/looped tracked-copy controls pass. The new sweep confirms ten further synthetic acceptances and one leak in an unchanged tracked script.

**Existing tracked example:** record.sh allocates BUILD_WORK at line 34, runs go build at line 36, and installs the EXIT trap only at line 46. Running its normal --verify entrypoint with a dedicated TMPDIR and `GOTOOLCHAIN=invalid` makes that build fail with exit 1. BUILD_WORK remains, and the scanner approves the actual unchanged file with no diagnostic. This is a pre-existing defect inside the new gate's explicit examples/ scope, rather than a hypothetical malicious script.

The minimal equivalent is also accepted:

```bash
set -euo pipefail
w="$(mktemp -d)"
false
trap 'rm -rf "$w"' EXIT
```

The trap and allocation are in the same branch, but cleanup is never installed. Replacing only the tracked migration-guides script temporarily with this fixture makes the **complete hygiene gate return 0**. The same acceptance occurs for an assertion or TERM before registration.

**Arithmetic tokenizer failure:** despite the stated arithmetic exclusion, the delimiter scanner treats the shift below as a heredoc and blanks the live allocation after it:

```bash
bit=1
x=$((1 << bit))
w="$(mktemp -d)"
```

This valid script exits 0 and leaves its directory. Its tracked-path substitution also makes the **complete hygiene gate return 0**. Quoted arithmetic expansion and the arithmetic command form reproduce the same failure.

All eleven executed leaks are recorded together:

| Accepted case | Result |
| --- | --- |
| false before trap registration | Exit 1, directory remains |
| Failed assertion before trap registration | Exit 1, directory remains |
| TERM before trap registration | Exit -15, directory remains |
| Arithmetic expansion with variable shift | Exit 0, allocation hidden by false heredoc recognition |
| Quoted arithmetic expansion with variable shift | Exit 0, same omission |
| Arithmetic command with variable shift | Exit 0, same omission |
| for-loop binding clears an allocated variable | Exit 0, root identity lost despite one textual assignment |
| Array-element assignment clears scalar root | Exit 0, mutation missed by the assignment regex |
| Visible rm function returns without deleting | Exit 0, literal trap text is mistaken for effective removal |
| read writes empty POSE_TEST_TMP after helper init | Exit 0, helper root identity overwritten without an equals-sign assignment |
| Existing record.sh build fails before its trap | Exit 1, BUILD_WORK remains |

Correct two-root and helper controls are accepted and leave nothing. A rejected syntactically invalid split-definition experiment is excluded from findings and counts. The own-trap cases use no sourced file or eval; the helper case sources only the shipped helper and visibly overwrites its root in the inspected script. The documented arbitrary-shell limitation therefore does not account for these ordinary paths.

Expected action: ensure accepted cleanup protects fallible work after allocation, including the actual demo build. Correct arithmetic/heredoc tokenization, binding mutation detection and command resolution for the supported subset, or reject unsupported forms conservatively. Add the reproduced cases and require both complete-gate counterexamples to fail. Keep the explicit eval/sourced-file boundary distinct from the accepted direct constructs.

## Previous findings and requirement judgment

F1, F3, F4, F5 and F6 remain resolved. Twelve independent helper cases cover nested 0500, 0400 and 0000 entries across success, assertion, set-e and unset-variable exits, all leaving TMPDIR empty. The exact shipped fixture reaches its setup marker and returns 0/1/1/1 as expected. Helper INT/TERM return 130/143 and clean up. Removing the INT handler fails the gate and leaves no sandbox. Composition instructions name pose_test_tmp_remove.

The hygiene command itself returns INT 130 and TERM 143 at the formerly failing permission-control point, leaving its caller TMPDIR empty. Reverting the helper permission command also makes the gate fail with exit 1 and clean up. DEBUG hooks select only signal timing; they do not alter fixture or cleanup code.

R1 helper/hygiene behavior passes this matrix, while the scan still approves the existing demo's R1 failure. R2 passes: all three real scripts return 0 and leave their own TMPDIR empty. Migration and install run from another directory; quickstart additionally passes there with an absolute --binary. Install also passes with --binary and --previous 6.2.0. GitHub works anonymously; no token was needed. All real-script and module-suite runs completed before temporary mutations.

R3 remains unsatisfied (F2). Restoring the three scripts from origin/main makes the static gate fail. The old quickstart returns 0 but leaves a tmp.* directory, which fails the nonempty-directory condition. The no-helper control works. These prove rejection of the original leak, while the accepted cases establish the current gap.

The full branch diff preserves the three changed scripts' assertions, source/init-before-first-mktemp order and later cd/argument handling. Quickstart's trailing success-only rm is replaced by EXIT cleanup. Exported TMPDIR captures ordinary Go work directories and allocated curl/git fixtures; none of the three scripts relies on preserving the caller's TMPDIR value. Every tracked *.sh currently lies in tests/, scripts/, examples/ or install.sh, and Git pathspecs include descendants. Helper and hygiene are exempt from static scanning and tested dynamically.

D1 remains proportionate: one owned root captures descendant outputs without maintaining a list. SIGKILL independently leaves its root, an unavoidable trap limitation to disclose before closeout; Final Report remains Pending. F2 is an observed defect, not accepted residual risk.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, the hygiene command and the explicit TestLocalVerifyCoversCurrentCIGates run pass. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene command. Passing commands do not override the failing requirement evidence.

Artifact-check exits 0: all eleven declared artifacts are observed; thirteen observed paths include the two earlier review artifacts, and no native finding names this spec. Implementation commits carry POSE-Spec attribution. Tech-debt assessment finds zero markers/uncovered items; recurrence-check flags zero keys. No public MCP/REST/Protobuf/Kafka contract changed; assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full output and executable probes remain outside the repository at /tmp/pose-temp-independent-r6; the manifest stores exit codes, facts, every finding case and SHA-256 hashes. Mutations ran sequentially after real-script checks and were restored byte-for-byte. Every deliberate leak was observed and removed. Assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-c7f6cabe03b0b57e`, unsealed. No sealed bundle or attestation was created. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because the medium finding prevents entering the conditional approval closeout. Tasks, trace, plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- Type: bugfix with CI/validation integration; .pose/workflows/review.md and pose-review/pose-spec-closeout skills.
- Security: temporary paths, permissions, signals, token handling and evidence integrity.
- Documentation style: spec, cleanup instructions and report accuracy.
- Backend Go: selected by the effective plan; branch build, vet and full tests exercised, with no Go-source change.
- Delivery evidence: distinguish check execution from semantic satisfaction.
- Knowledge governance: consulted; no accepted residual risk or handoff.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2 against the supported plain-shell forms, including the actual tracked demo failure. Keep the spec open and make the complete-gate failure-before-trap and arithmetic-shift controls fail before any seal or closeout.
