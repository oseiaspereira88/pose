# Independent review, round 5: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `8d0b3370cbae0f131855e544a1e848df25f3a8e2`. Exhaustive sweep completed before deciding: F2 remains a medium finding. F6 is resolved and no separate new finding was identified. No persistent implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied and not independently authenticated; the prefix does not establish signed execution provenance.

Reviewed the complete current branch diff, correction diff, spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. Previous-round records remain in commit 8d0b3370; these owned review artifacts now describe round 5.

## Complete findings

### F2 — Medium: ordinary accepted shell still leaves temporary roots

Location: `tests/hygiene/temp-cleanup.sh:93`, with related defects at lines 40–59, 70–81, 111–114, 131, 157 and 162. Contract: **R3**.

The current code correctly rejects all 26 external leaking fixtures retained from rounds 3 and 4. Its 35 negative fixtures, 5 positive fixtures and rebound tracked-copy control also pass. The new sweep nevertheless confirms thirteen further accepted leaking scripts, including plain control flow:

```bash
if false; then
exit 0
fi
w="$(mktemp -d)"
```

The scanner cuts the file at any column-zero exit/return before computing nesting. Here that exit never runs, but the live allocation is omitted and the scan reports success. Substituting this script into the already tracked migration-guides path makes the **complete hygiene gate return 0**.

A second complete-gate counterexample is:

```bash
for i in 1 2; do
  w="$(mktemp -d)"
  trap 'rm -rf "$w"' EXIT
done
```

The assignment occurs once in the text and twice at runtime. The trap removes only the last directory. The scanner accepts the matching block IDs; both the fixture and the complete gate return 0 while the first root remains.

The complete executed corpus is below. Every row was accepted; twelve leave the assigned parent TMPDIR nonempty and the absolute-template case leaves a directory outside the helper's owned root.

| Accepted case | Concrete failure |
| --- | --- |
| Exit in skipped branch | Truncates inspection before a live allocation |
| Allocation in then, trap in else | Both arms receive the same block identity although only the allocation runs |
| Repeated loop allocation | One syntactic binding produces multiple roots |
| Function-local allocation | EXIT runs after the local variable is gone; exits 1 and leaves the root |
| Function called twice | One syntactic binding allocates twice and loses the first root |
| Cleanup definition inside skipped branch | Definition is counted but never executed; EXIT returns 127 and leaks |
| Allocation/trap in different case arms | Shared enclosing case identity is mistaken for shared execution |
| Helper trap with false before remove | Under set-e, the trap exits 1 before reaching the required final cleanup |
| Cleared POSE_TEST_TMP | Helper root identity can be overwritten despite TMPDIR being protected |
| Helper with absolute mktemp template | mktemp ignores TMPDIR and its root survives helper cleanup |
| Quoted text containing <<EOF | A printed string is mistaken for a heredoc and hides later code |
| Inline comment containing <<EOF | Comment text is mistaken for a heredoc and hides later code |
| Unquoted hyphenated heredoc delimiter | The delimiter is parsed as a word prefix and the remainder of the file is hidden |

Ten leaky fixtures exit 0; the remaining three exit 1, 1 and 127 and still leave their files, which R1/R3 must also cover. Correct two-root trap and helper controls pass and leave nothing.

The spec now accurately calls static analysis a regression guard and identifies eval/sourced-file limits. The failures above use direct ordinary shell constructs. The own-trap counterexamples use no sourced file or eval; helper examples source only the shipped helper and put the failing behavior visibly in the inspected script. The stated boundary therefore does not explain these acceptances. The stronger statement that other unsupported forms are rejected is still unsupported by execution evidence.

Expected action: make reachability, branch/iteration identity, allocation lifetime, helper cleanup invariants and heredoc tokenization sound for the accepted subset, or reject those forms. Require the two complete-gate negative controls to fail. Keep arbitrary-shell limitations distinct from the direct constructs the guard claims to enforce.

## Previous findings and requirement judgment

F1, F3, F4 and F5 remain resolved. Independent helper tests cover nested 0500, 0400 and 0000 entries across success, assertion, set-e and unset-variable exits; all twelve leave TMPDIR empty. The exact shipped fixture reaches its setup marker in all four scenarios and returns 0/1/1/1 as expected. Direct helper INT/TERM probes return 130/143 and clean up. Removing the INT handler makes the gate fail and leave its caller TMPDIR empty. Composition instructions still name the real remove function.

F6 is resolved: delivering INT/TERM at the previously failing permission-control boundary now returns 130/143 and leaves the caller TMPDIR empty. Reverting only the helper chmod makes the gate fail with exit 1 and also leaves no sandbox. External DEBUG hooks select signal timing without changing fixture or cleanup code.

R1's helper and hygiene behaviors pass the independent failure/permission/signal matrix. R2 passes: all three real scripts return 0 and leave independent TMPDIR roots empty. Migration and install run from another directory; quickstart also passes there with an absolute --binary. Install additionally passes with --binary and --previous 6.2.0. GitHub works anonymously; no token was needed. All real scripts and module tests completed before temporary tracked-file mutations began.

R3 remains unsatisfied (F2). Restoring the three scripts from origin/main makes the static gate fail. The old quickstart returns 0 but leaves a tmp.* directory, failing the nonempty-directory condition. The no-helper control is effective. These checks prove rejection of the original defect, while the new cases establish the remaining enforcement gap.

The branch diff preserves the three real-script assertions, source/init-before-first-mktemp ordering and later cd/argument behavior. Quickstart's old success-only rm is replaced by EXIT cleanup. Exported TMPDIR captures ordinary Go work directories and allocated curl/git fixtures; none of the three scripts relies on retaining the caller's TMPDIR value. Every tracked *.sh currently lies within tests/, scripts/, examples/ or install.sh; Git pathspecs include descendants. Helper and hygiene are exempt from the static scan and exercised dynamically.

D1 remains proportionate: one owned root captures descendant outputs without maintaining a list. SIGKILL independently leaves its root, an unavoidable trap limitation to disclose before closeout; the Final Report remains Pending. F2 is an observed defect rather than an accepted residual risk.

## Checks and compact evidence

The branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, the hygiene command and explicit TestLocalVerifyCoversCurrentCIGates run pass. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene command. These passing commands do not override the failing requirement evidence.

Native artifact-check exits 0: all eleven declared artifacts are observed. Thirteen observed paths include the two earlier review artifacts, with no native finding naming this spec. Implementation commits carry POSE-Spec attribution. Tech-debt assessment finds zero markers/uncovered items; recurrence-check flags zero keys. No public MCP/REST/Protobuf/Kafka contract changed, so assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full outputs and executable probe sources remain outside the repository at /tmp/pose-temp-independent-r5; the manifest contains exit codes, facts, all finding cases and SHA-256 hashes. Mutations ran sequentially after real-script checks and were restored byte-for-byte. Every deliberate leak was observed and removed. Native assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-b0708e511ddd5bda`, unsealed. No sealed bundle or attestation was created. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because the medium finding prevents entering the conditional approval closeout. Tasks, trace, implementation plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- Type: bugfix with CI/validation integration; workflow .pose/workflows/review.md and skills pose-review/pose-spec-closeout.
- Security: temporary paths, permissions, signals, token handling and evidence integrity.
- Documentation style: spec, cleanup instructions and report accuracy.
- Backend Go: selected by the effective plan; branch build, vet and full tests exercised, with no Go-source change.
- Delivery evidence: distinguish check execution from semantic satisfaction.
- Knowledge governance: consulted; no residual risk accepted or handoff created.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2 against the stated accepted shell subset, then request a fresh independent review. Keep the spec open and make both complete-gate counterexamples fail before any seal or closeout.
