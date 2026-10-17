# Independent review, round 4: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `06b03c7025b28d00056b660ecdf90f1790c79c99`. Full sweep completed before deciding: F2 remains open and F6 is a new medium finding. F5 is resolved. No persistent implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. The maintainer-supplied individual implementer/session identity is not independently authenticated; the prefix does not prove signed execution provenance.

Reviewed the complete current branch diff, correction diff, spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. Previous-round review files remain in commit 06b03c70; these owned files now report round 4.

## Complete findings

### F2 — Medium: the claimed restricted grammar still accepts leaks

Location: `tests/hygiene/temp-cleanup.sh:77`, also lines 57–68, 73, 88–95. Affected contract: **R3**. The twenty built-in negatives improve coverage but do not establish that only the documented safe forms pass.

This ordinary variable-reuse regression is accepted:

```bash
w="$(mktemp -d)"
w="$(mktemp -d)"
trap 'rm -rf "$w"' EXIT
```

Both allocations match the accepted whole-assignment form. `assigned[var] = line` discards the first binding, while runtime cleanup only removes the final value. The fixture exits 0 and leaves the first directory. Substituting it into the already tracked migration-guides script makes the **complete hygiene gate return 0**.

All fourteen new leaking fixtures were accepted by the exact production scanner and exited 0. Thirteen left their assigned TMPDIR nonempty; one unset TMPDIR after initializing the helper and left its allocation in the default temporary directory. The full set is:

| Mechanism | Independently executed cases |
| --- | --- |
| Lost allocation binding | Reuse a variable for a second mktemp; clear the variable after allocation |
| Cleanup is never executed | Trap inside `if false`; `exit 0` before rm in the trap; `return` before rm in its function |
| Cleanup replaced | Redefine the cleanup function; override with `builtin trap`; override the helper with `builtin trap` |
| Helper protection only resembles valid code | Init inside `if false`; EXIT trap merely echoes `pose_test_tmp_remove`; redefine remove using Bash `function name { ...; }` without parentheses |
| Source text mistaken for execution | Helper source/init in a heredoc; an own trap in a heredoc |
| Allocation bypasses the owned root | Unset TMPDIR after helper initialization |

Correct two-root trap and helper controls also passed and left nothing. These are observed behavior failures, not hypothetical parser limitations. In particular, statements other than rm are ignored rather than rejected inside trap bodies, “top-level” is not established by a column-zero match, and helper trap composition still uses a name-substring check.

Expected action: enforce effective control flow, cleanup lifetime and allocation bindings for whatever restricted form is accepted, or use execution-based leak evidence for accepted scripts. Reject unsupported constructs conservatively. Cover the reproduced cases and require the full-gate repeated-binding control to fail. Do not claim arbitrary other shapes are rejected on the evidence of twenty fixtures.

### F6 — Medium: the hygiene script leaks its own restricted fixtures on interruption or failure

Location: `tests/hygiene/temp-cleanup.sh:21`, exposed by the permission control at lines 180–183. Affected contract: **R1**, including failure, INT/TERM and restricted entries.

The control deliberately runs the defective helper, leaving 0000/0400 entries until line 183 repairs permissions. The hygiene script's own EXIT trap still uses `chmod -R u+w`, the same insufficient operation corrected in the helper. Delivering **INT or TERM immediately before that repair** runs EXIT cleanup but prints rm Permission denied and leaves the sandbox in the caller TMPDIR. The unchanged branch script reproduced this for both signals, exiting -2/-15 respectively. An external BASH_ENV DEBUG hook only selected the instant to deliver the signal; it changed no cleanup, fixture or script code.

The independent old-chmod mutation now correctly makes the gate fail (exit 1), resolving F5, but the failing command also leaves its sandbox for the same reason. This is especially relevant for a hygiene gate designed to exercise failure: its failure path must not recreate the leak. The static scan exempts this script, so that check cannot detect its own defective trap.

Expected action: make the hygiene script remove its complete sandbox on assertion failure and interruption, restoring directory traversal before removal. Add failure and INT/TERM controls for the hygiene command itself and assert an empty caller TMPDIR, including during the deliberately leaky permission control.

## Previous findings and requirement judgment

F1, F3 and F4 remain resolved. Independent helper fixtures cover nested 0500, 0400 and 0000 entries across success, assertion, set-e and unset-variable exits; all twelve leave TMPDIR empty. Direct INT/TERM probes return 130/143 promptly and clean up. Removing only the INT handler makes the full gate fail. Composition documentation still names the real remove function.

F5 is now resolved: the exact shipped body reaches its setup marker, returns the expected success 0 or failure 1 for assertion/set-e/unset-variable cases, and cleans up. The old-chmod regression returns 1 rather than passing. The new failure-path cleanup defect is separately classified as F6.

R1 helper behavior passes, but the new hygiene script itself violates R1 under the reproduced interruptions (F6). R2 passes: all three real scripts return 0 and leave independent TMPDIR roots empty. Migration and install ran from another directory, quickstart additionally ran from another directory with an absolute --binary, and install passed with explicit --binary and --previous 6.2.0. GitHub worked anonymously; no token was needed. An initial migration run overlapped the tracked-script substitution, was explicitly discarded, and was replaced after restoration; only the valid run is counted.

R3 is not satisfied (F2). Restoring the three scripts from origin/main makes the static gate fail. Running the old quickstart returns 0 but leaves a tmp.* directory, which fails the nonempty-directory check. The built-in no-helper control and twenty negative/five positive scan fixtures run, but do not cover the remaining acceptances.

The full branch diff preserves script assertions and source/init-before-first-mktemp ordering. Later cd/argument handling is unchanged; the quickstart's old success-only rm is replaced by EXIT cleanup. Exported TMPDIR contains Go work directories and allocated curl/git fixtures, and none of the three scripts depends on retaining the caller's TMPDIR value. Every tracked *.sh currently lies within tests/, scripts/, examples/ or install.sh; Git pathspecs cover nested scripts. Helper and hygiene are explicitly exempted from the static scan; F6 lies in the latter.

D1 remains proportionate: a single owned root captures descendant outputs without maintaining a list. The independent SIGKILL probe still leaves its root, an unavoidable trap limitation. State that boundary honestly before closeout; the Final Report remains Pending. F2/F6 are defects, not accepted residual risks.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, the hygiene command and the explicit TestLocalVerifyCoversCurrentCIGates run passed. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene command. Passing checks do not override the failing requirement evidence.

Native artifact-check exited 0: all eleven declared artifacts are observed. Thirteen observed paths also include the two earlier review artifacts; no native finding names this spec. Implementation commits carry POSE-Spec attribution. Tech-debt assessment found zero markers/uncovered items; recurrence-check flagged zero keys. No public MCP/REST/Protobuf/Kafka contract changed; assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Complete output and probe sources remain outside the repository at /tmp/pose-temp-independent-r4; the manifest stores exit codes, facts, findings and SHA-256 hashes. Temporary source mutations were restored byte-for-byte and deliberately leaked roots were observed then removed. Native assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-46959ff7c2d2f8a4`, unsealed. No sealed bundle or attestation was created. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because medium findings prevent entering the conditional approval closeout. Tasks, trace, plan and roadmaps remain unchanged. No commit/push or upstream contribution was made.

## Rules applied during review

- Type: bugfix with CI/validation integration; workflow .pose/workflows/review.md and skills pose-review/pose-spec-closeout.
- Security: temporary paths, permissions, signals, token handling and evidence integrity.
- Documentation style: spec, cleanup instructions and report accuracy.
- Backend Go: selected by the effective plan; branch build, vet and full tests exercised, with no Go-source change.
- Delivery evidence: distinguish check execution from semantic satisfaction.
- Knowledge governance: consulted; no accepted residual risk or handoff artifact.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2 and F6 and request a fresh independent review. Keep the spec open. Prove both full-gate adversarial acceptance and failure/signal cleanup before any seal or closeout.
