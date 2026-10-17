# Independent review, round 7: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `f46f04cabf752d3bd81658c1bdc3975f2eba3162`. Exhaustive sweep completed before deciding: F2 remains a medium finding, with five further accepted executed leaks. The existing demo leak is resolved. No persistent implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied and not independently authenticated; the prefix does not prove signed execution provenance.

Reviewed the full branch diff, correction diff, spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. Prior-round records remain in commit f46f04ca; these owned artifacts now report round 7.

## Complete findings

### F2 — Medium: accepted forms still let failure bypass cleanup

Location: `tests/hygiene/temp-cleanup.sh:225` and `:228`, with related lexical state at `:81` and trap operands at `:248–252`. Contract: **R3**.

All 49 prior external synthetic leak fixtures are now rejected. The corrected demo leaves TMPDIR empty on its normal --verify path and when GOTOOLCHAIN=invalid makes the build fail. The 67 built-in negatives, 5 positives and three tracked-copy mutation controls pass. The new sweep still confirms five accepted leaking scripts, all clean under **ShellCheck --severity=warning**.

**The new adjacency check discards the very contents it must classify.** It runs masked() before checking a plain assignment, so both a command substitution and a potentially fatal expansion become the same `x=""` text as a harmless assignment. This fixture is accepted:

```bash
set -euo pipefail
w="$(mktemp -d)"
x="$(false)"
trap 'rm -rf "$w"' EXIT
: "$x"
```

It exits 1 before the trap is registered and leaves the directory. Replacing the already tracked migration-guides path temporarily with this fixture makes the **complete hygiene gate return 0**. A variable assigned only in an unexecuted optional branch also fails under set-u when expanded in the gap and leaves the root.

**Arithmetic state is still confined to one line.** Counting opening/closing arithmetic tokens in the current line fixes the previous one-line shifts, but a valid multiline shift is interpreted as a heredoc:

```bash
bit=1
x=$((1
 << bit))
: "$x"
w="$(mktemp -d)"
: "$w"
```

The scanner blanks the live allocation after the shift, reports success, and the script returns 0 while leaving its directory. The quoted expansion form reproduces it. The unquoted fixture also makes the **complete hygiene gate return 0** in the tracked-path substitution.

**An allowed rm operand can abort the whole trap.** A cleanup body containing `rm -rf "$w" "$OPTIONAL"` is accepted even if OPTIONAL was initialized only in an unexecuted branch. Under set-u, expansion fails before rm runs, leaving w. ShellCheck recognizes the syntactic assignment and emits no warning, so the existing warning gate does not compensate.

All finding cases, together:

| Accepted case | Exit | Observed result |
| --- | --- | --- |
| Command substitution fails in an apparently plain assignment before the trap | 1 | TMPDIR nonempty |
| Unset conditional variable expanded before the trap | 1 | TMPDIR nonempty |
| Unset conditional variable expanded as an additional trap operand | 1 | TMPDIR nonempty |
| Multiline arithmetic shift | 0 | Later allocation hidden; TMPDIR nonempty |
| Quoted multiline arithmetic shift | 0 | Same omission; TMPDIR nonempty |

Three correct controls (two-root trap, simple path assignment, helper) are accepted and leave nothing. An arithmetic-error experiment that leaves no residue is excluded from findings. No finding relies on eval or hidden sourced-file behavior. These cases contradict the stated accepted-form checks or the rule that unsupported syntax is rejected.

Expected action: inspect expansion and command-substitution effects before permitting a plain assignment in the unprotected gap. Ensure permitted cleanup operands cannot abort removal under nounset. Maintain arithmetic/quote context across lines, or reject unsupported multiline syntax before declaring that no allocation exists. Make all five cases and both complete-gate controls fail.

## Previous findings and requirement judgment

F1, F3, F4, F5 and F6 remain resolved. Twelve independent helper cases cover nested 0500, 0400 and 0000 entries across success, assertion, set-e and unset-variable exits; all leave TMPDIR empty. The exact shipped body reaches its setup marker and returns 0/1/1/1. Helper INT/TERM return 130/143 and clean up; removing INT handling makes the gate fail. Composition instructions name the real remove function.

The hygiene command returns INT 130 and TERM 143 at the former permission-control failure point and leaves its caller TMPDIR empty. Reverting the helper chmod makes the gate fail with exit 1 and also clean up. DEBUG hooks only choose signal timing.

The demo's build-failure leak is resolved: allocations/trap now precede the build, its artifact claim was added, and no assertion changed. The demo passes normally and the induced build failure returns 1 with no residue.

R1 helper/hygiene behavior passes the independent matrix; R2 passes for the three real scripts with separate TMPDIR roots. Migration and install run from another directory; quickstart additionally passes there with an absolute --binary. Install also passes with --binary and --previous 6.2.0. GitHub works anonymously; no token was needed. Real-script and module-suite runs completed before mutations.

R3 remains unsatisfied (F2). Restoring the three scripts from origin/main makes the static gate fail. The old quickstart returns 0 but leaves a directory, failing the nonempty-directory check. The no-helper control works. These establish rejection of the original defect, while the five cases establish the remaining gap.

The three changed real-script assertions and source/init/cd/argument behavior are preserved. Quickstart's success-only rm is replaced by EXIT cleanup. TMPDIR captures ordinary Go work directories and allocated curl/git fixtures, with no dependency in these scripts on retaining the caller's original value. All tracked *.sh are currently in tests/, scripts/, examples/ or install.sh; Git pathspecs cover descendants. Helper/hygiene are exempt from static scanning and checked dynamically.

D1 remains proportionate: a single owned root captures descendant outputs without maintaining a list. SIGKILL independently leaves its root, an unavoidable trap limitation requiring disclosure before closeout; Final Report is still Pending. F2 is a defect, not accepted residual risk.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, hygiene and TestLocalVerifyCoversCurrentCIGates pass. All five adversarial leaking scripts separately pass the same ShellCheck severity. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene command.

Artifact-check exits 0: all twelve declared claims are observed. Fourteen observed paths include the two earlier review artifacts, with no native finding naming this spec. Implementation commits carry POSE-Spec attribution. Tech-debt assessment finds zero markers/uncovered items; recurrence-check flags zero keys. No MCP/REST/Protobuf/Kafka contract changed; assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full output and executable probes remain outside the repository at /tmp/pose-temp-independent-r7; the manifest contains exit codes, facts, all finding cases and SHA-256 hashes. Temporary mutations ran sequentially and were restored byte-for-byte. All deliberate leaks were observed and removed. Native assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-ccf11365ecce949e`, unsealed. No sealed bundle or attestation was created. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because the medium finding prevents entering the conditional approval closeout. Tasks, trace, plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- Type: bugfix with CI/validation integration; .pose/workflows/review.md and pose-review/pose-spec-closeout skills.
- Security: temporary paths, permissions, signals, token handling and evidence integrity.
- Documentation style: spec, cleanup instructions and report accuracy.
- Backend Go: selected by the effective plan; branch build, vet and full tests exercised, with no Go-source change.
- Delivery evidence: distinguish successful checks from requirement satisfaction.
- Knowledge governance: consulted; no accepted residual risk or handoff.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2 for the claimed accepted forms, prove the five negative cases and both complete-gate controls reject them, and request a fresh independent review before any seal or closeout.
