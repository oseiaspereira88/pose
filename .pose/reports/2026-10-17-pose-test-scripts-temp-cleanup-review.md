# Independent review, round 9: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `f98d0056fe399a51230d7df7719d78a7afb4d6f1`. Exhaustive sweep completed before deciding: F2 remains a medium finding, with four accepted leaking scenarios spanning two mechanisms. One occurs in the unchanged current demo. No persistent implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied and not independently authenticated; the prefix does not establish signed execution provenance.

Reviewed the full branch diff, correction diff, spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. Previous-round records remain in commit f98d0056; these owned artifacts now report round 9.

## Complete findings

### F2 — Medium: later allocation failure is unprotected; readonly option lists are only partly inspected

Locations: `tests/hygiene/temp-cleanup.sh:239` (skipping allocation lines before trap registration), `:183` (readonly predicate), and `examples/demo/record.sh:34–36`. Contracts: **R3**, and **R1** for the current demo's failure path.

All 59 previous external synthetic leak cases are now rejected. The built-in 78 negative fixtures, 5 positives and three tracked-copy mutation controls pass. Four further accepted scenarios still leave their dedicated TMPDIR nonempty.

**1. Adjacent allocations can fail before cleanup is registered.** The accepted two-root shape installs its only trap after both mktemp commands succeed. Failure of the second command leaves the first root, even without another command in the gap:

```bash
set -euo pipefail
a="$(mktemp -d)"
b="$(mktemp -d x)"
trap 'rm -rf "$a" "$b"' EXIT
```

The second template is invalid, so this concrete fixture exits 1 and leaves a. The exact scanner approves it, and substituting it into the tracked migration-guides path makes the **complete hygiene gate return 0**. This is an allocation-failure path, rather than hidden eval/source behavior.

The same flaw exists in the **current tracked demo**, which performs BUILD_WORK=mktemp, WORK=mktemp, then installs the trap. Controlled fault injection delegates its first call to the native mktemp and returns exit 1 only on the second. The unchanged demo exits 1 and leaves BUILD_WORK. The counter confirms exactly two calls, no demo source is changed, and the scanner approves the actual file. This models an ordinary allocation error; no signal race or arbitrary-shell trick is needed. The previous go-build error is cleaned up because it occurs after registration, but failure during the allocation sequence remains uncovered.

The existing two-root positive fixture therefore demonstrates happy-path cleanup only. Cleanup must already protect earlier roots when a subsequent allocation fails.

**2. Readonly detection only checks the first declaration option word.** `declare -x -r label="old"` and `declare -x -r TMPDIR` are valid declarations, but the predicate expects r in the first option token after declare. Both pass. The first makes an accepted literal assignment in the unprotected gap fail; the second makes helper initialization allocate its root and then abort when assigning readonly TMPDIR before registering traps. Both return 1 and leave residues. The readonly option itself is present visibly in the inspected text.

All finding scenarios:

| Accepted scenario | Exit | Observed result |
| --- | --- | --- |
| Second allocation has an invalid template | 1 | First root remains |
| Current demo: only its second mktemp fails | 1 | BUILD_WORK remains |
| declare -x -r label followed by literal assignment before trap | 1 | Root remains |
| declare -x -r TMPDIR before helper initialization | 1 | Helper root remains |

The readonly-helper fixture also makes the **complete hygiene gate return 0** when placed temporarily in the tracked migration path. All three synthetic leaks pass ShellCheck at CI warning severity; the unchanged demo passes the normal CI glob check. A typeset readonly-target probe is rejected by the variable-mutation rule and is excluded from accepted-leak findings. Three ordinary successful controls clean up on their normal runs; that does not establish protection against later allocation failure.

Expected action: make cleanup active before subsequent allocations can fail, or reject the unsafe own-trap shape until protected; validate allocation-N failure in the demo and multi-root fixture. Parse all declaration option tokens when enforcing the readonly prohibition. Require both complete-gate counterexamples to fail or produce verified empty-directory cleanup.

## Previous findings and requirements

F1, F3, F4, F5 and F6 remain resolved in their reviewed scope. Twelve helper cases cover nested 0500, 0400 and 0000 entries across success, assertion, set-e and unset-variable exits; all leave TMPDIR empty. The exact shipped body reaches its setup marker and returns 0/1/1/1. Helper INT/TERM return 130/143 and clean up; removing INT handling makes the gate fail and clean up. Composition instructions name pose_test_tmp_remove.

The hygiene command returns INT 130 and TERM 143 at the former permission-control failure point and leaves no sandbox. The old-chmod mutation fails with exit 1 and also cleans up. DEBUG hooks select only signal timing. These checks use mutable helper state; F2 identifies the unsupported readonly contexts still admitted by the guard.

The demo passes normal --verify and its GOTOOLCHAIN=invalid build error returns 1 with an empty TMPDIR. Its second-allocation failure, however, independently falsifies R1. R2 passes for the three real scripts with their own TMPDIR roots; migration and install run from another directory, quickstart additionally passes there with an absolute --binary, and install passes with --binary and --previous 6.2.0. GitHub works anonymously; no token was needed. All real-script and module-suite runs completed before mutations.

R3 remains unsatisfied (F2). Restoring the three scripts from origin/main makes the scan fail; the old quickstart returns 0 but leaves a directory, failing the nonempty-directory condition. The no-helper control works. These prove rejection of the original defect while the new cases establish the remaining gap.

The full diff preserves assertions and source/init/cd/argument behavior of the three real scripts and assertions of the demo. TMPDIR captures ordinary Go work directories and allocated curl/git fixtures, with no reliance on retaining the caller's original value. All tracked *.sh currently fall under tests/, scripts/, examples/ or install.sh, with descendants included by Git pathspecs. Helper/hygiene are statically exempt and dynamically checked.

D1 remains proportionate: a single helper-owned root registered before subsequent allocations avoids this multi-root cleanup gap. SIGKILL independently leaves its root, an unavoidable trap limitation requiring disclosure before closeout; Final Report remains Pending. F2 is an observed defect, not accepted residual risk.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, hygiene and TestLocalVerifyCoversCurrentCIGates pass. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene command. The synthetic counterexamples separately pass the warning check.

Artifact-check exits 0: all twelve declared claims are observed. Fourteen observed paths include the two prior review artifacts, with no native finding naming this spec. Implementation commits carry POSE-Spec attribution. Tech-debt assessment finds zero markers/uncovered items; recurrence-check flags zero keys. No MCP/REST/Protobuf/Kafka contract changed; assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full outputs and executable probes remain outside the repository at /tmp/pose-temp-independent-r9; the manifest stores exit codes, facts, every finding scenario and SHA-256 hashes, including the mktemp fault-injection shim. Mutations ran sequentially and were restored byte-for-byte. All intentional leaks, including the rejected negative-control leak, were observed then removed. Native assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-177ef5134d45caa8`, unsealed. No sealed bundle or attestation was created. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because the medium finding prevents entering the conditional approval closeout. Tasks, trace, plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- Type: bugfix with CI/validation integration; .pose/workflows/review.md and pose-review/pose-spec-closeout skills.
- Security: temporary paths, permissions, signals, token handling and evidence integrity.
- Documentation style: spec, cleanup instructions and report accuracy.
- Backend Go: selected by the effective plan; branch build, vet and full tests exercised, with no Go-source change.
- Delivery evidence: distinguish successful commands from requirement satisfaction.
- Knowledge governance: consulted; no accepted residual risk or handoff.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2, including the current demo's allocation-failure path, and make both complete-gate controls reject or safely clean the demonstrated cases before another independent review or any seal/closeout.
