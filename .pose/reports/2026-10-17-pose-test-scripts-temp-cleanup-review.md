# Independent review, round 10: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `e5d86dfb47a150335a2a64e06c7af6dae6fa7796`. Exhaustive sweep completed: two medium findings, F2 and F7. No implementation fix, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, maintainer authorized. This is Codex/OpenAI reviewing a Claude/Anthropic implementation, a different vendor. The `agent:independent-` prefix is an engine-required declaration until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied, not independently authenticated.

Reviewed the full branch diff, current correction, spec, AGENTS.md, POSE review/closeout instructions, applicable rules and pose-review/pose-spec-closeout skills. Built and used the requested branch CLI. No Requirements or Technical Plan edits were made. These reviewer-owned artifacts replace the round-9 account retained in commit e5d86dfb.

## Complete findings

### F2 — Medium: the guard still approves visible plain-shell programs that leak

Locations: `tests/hygiene/temp-cleanup.sh:162` (physical-line exit/return cutoff), `:51` and `:94` (quote/substitution state reset per line), `:155` and `:167` (accepting files after allocations are hidden), `:272` (own-trap shape without checking disabling state). Requirement: **R3**.

All 63 previously reproduced external synthetic leaks are now rejected. The shipped 81 negative fixtures, 5 positives and 3 tracked-copy controls pass. Eleven further programs are accepted by the exact scanner and independently leave a root in their own TMPDIR. Substituting each into tracked tests/import/migration-guides.sh also makes the **complete hygiene gate return 0**. The gate's own TMPDIR is empty; these full-gate controls prove admission, while separate executions prove the substituted scripts leak.

**Reachable code is discarded at an exit/return text line.** The block tracker does not track command substitutions or line continuations, and the cutoff also matches function declarations. For example, this CI-warning-clean script returns 0, allocates a directory and never removes it, but the scanner discards everything after the exit line:

```bash
set -euo pipefail
printf '%s\n' "$(
exit
)"
w="$(mktemp -d)"
: "$w"
```

The exit ends only the substitution. The same false cutoff occurs with printf followed by a backslash-newline and an argument exit or return; with exit()/return() function declarations; and with a bare top-level return in a script without errexit, where return reports an error and execution continues. A legacy-backtick substitution reproduces the same context loss.

**A multiline string starts a fictitious heredoc.** The lexer forgets that the previous line opened a quote, treats the following << END as code, then blanks the real allocation through EOF. This also returns 0, leaks, passes ShellCheck warning severity and passes the complete gate:

```bash
set -euo pipefail
printf '%s\n' '
<< END
'
w="$(mktemp -d)"
: "$w"
```

Both single and double quotes reproduce it. No eval or sourced file hides code in these cases; multiline quoting, continuation and substitution are direct shell syntax. If unsupported, the guard must reject the script rather than silently approve it as containing no allocation.

**An accepted own trap need not run successfully.** A single allocation followed immediately by the approved rm trap still leaks when a subsequent set -n disables execution, including the EXIT handler. A second ordinary case creates a/b/f and chmods a to 0000 after installing the trap: rm -rf returns 1 and leaves the root. Both pass warning-level ShellCheck and the full gate. The latter is the same restricted-directory failure class the helper deliberately handles, now admitted by the alternative cleanup form.

Every accepted leaking case (all have nonempty caller TMPDIR and complete-gate exit 0):

| Case | Script exit | ShellCheck warning exit |
| --- | --- | --- |
| `exit-function-noop` | 0 | 0 |
| `return-function-noop` | 0 | 0 |
| `bare-return-without-errexit` | 0 | 0 |
| `exit-as-continued-argument` | 0 | 0 |
| `return-as-continued-argument` | 0 | 0 |
| `multiline-single-quoted-heredoc-text` | 0 | 0 |
| `multiline-double-quoted-heredoc-text` | 0 | 0 |
| `multiline-backtick-return` | 0 | 1 |
| `command-substitution-exit` | 0 | 0 |
| `own-trap-noexec` | 0 | 0 |
| `own-trap-rm-readonly-child` | 1 | 0 |

Expected action: fail closed for unsupported lexical/execution contexts; preserve quote/continuation/substitution context before treating a line as heredoc or top-level termination. Reject or protect noexec and permission-changing own-trap forms. Cover the complete list through both the exact scan and tracked-path gate, or clean them up demonstrably. An arbitrary-shell proof is unnecessary: the admitted visible cases must not be mistaken for the two supported forms.

### F7 — Medium: a relative caller TMPDIR loses its meaning after cd

Locations: `tests/lib/tempdir.sh:13`, `:17`, `:28–29`; integration callers initialize before changing directory at tests/quickstart/first-governed-loop.sh:19, tests/import/migration-guides.sh:17 and tests/journeys/install-and-upgrade.sh:16. Requirements: **R1**, **R2**, and the constraint preserving caller TMPDIR behavior.

Start in a directory containing tmp, set TMPDIR=tmp and call the helper. GNU mktemp accepts that valid relative directory and returns tmp/tmp.<suffix>. The helper keeps that relative value as POSE_TEST_TMP and exports it as TMPDIR. After cd, allocations and removal resolve it relative to the new directory. chmod errors are suppressed and rm -rf succeeds against a nonexistent wrong path, leaving the actual root.

Independent helper reproduction: source the current helper, pose_test_tmp_init, cd /, true. Exit is 0 and the original caller tmp contains the root. All three current real scripts, invoked by absolute script path from a different directory with TMPDIR=tmp, exit 1 on their next mktemp and leave the new helper root behind. Install was given an absolute --binary and --previous 6.2.0; the failure precedes network use. There is no signal, readonly mutation or hidden source trick. Absolute TMPDIR runs pass, explaining why the shipped cases miss this.

Expected action: stabilize the allocated root as an absolute path before exporting it or using it for cleanup. Prove cleanup across cd and each actual script with relative caller TMPDIR, including failure exits.

## Corrections and requirement judgment

The round-9 corrections work: the unchanged demo makes one mktemp call; fault injection on a hypothetical second call is never reached, --verify returns 0 and TMPDIR is empty. GOTOOLCHAIN=invalid returns 1 with empty TMPDIR. Four previous synthetic forms, including stacked readonly flags and the second-allocation failure, are rejected; both prior complete-gate controls now return 1. Assertions remain unchanged.

F1, F3, F4, F5 and F6 remain resolved in their reviewed scope. Twelve independent helper permission/failure scenarios and the exact shipped setup (marker reached, exits 0/1/1/1) clean up with absolute TMPDIR. INT/TERM return 130/143 with default signal dispositions and empty TMPDIR. Removing the INT handler or reverting chmod makes the gate fail, and the gate still cleans up. Its own INT/TERM controls also return 130/143 with no residue. Documentation correctly names pose_test_tmp_remove.

R1 and R2 pass their absolute-TMPDIR matrices but fail F7. R3 remains unsatisfied by F2. Restoring the three origin/main scripts makes the static check fail; the old quickstart exits 0 and leaves a directory, failing the empty-TMPDIR condition. The no-helper control leaves a directory as expected.

D1 is proportionate: one owned TMPDIR captures child temporary work, and the demo now follows a single-root sequence. It still needs a stable absolute path. Ordinary Go build/test, curl and git usage has no dependency on retaining the caller's original TMPDIR value. SIGKILL independently exits -9 and leaves a root, an unavoidable trap limit that must be disclosed at closeout; Final Report remains Pending. Eval/source opacity is already declared; it does not account for F2's visible syntax and state changes. Findings are not accepted residual risks.

All 24 tracked *.sh files fall under the declared scan paths; Git pathspecs include descendants. Helper/hygiene are statically exempt and dynamically tested. No assertion or public API/MCP contract changed. Source/init precede allocations and cd, but the relative-path consequence is F7.

## Checks and evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over CI globs, the hygiene gate and TestLocalVerifyCoversCurrentCIGates pass. The three real scripts with absolute caller TMPDIR pass and leave it empty; migration and install run from another directory, quickstart additionally passes there with --binary, and install with --binary/--previous 6.2.0. No token was needed or exposed.

CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised gate. Artifact-check exits 0 with 12 claims observed among 14 paths, the two extras being review artifacts, and no finding for this spec. Tech-debt finds zero markers/uncovered items; recurrence flags zero keys. No inter-module contract change requires assess integrate.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full command outputs and executable probes are outside the repository in /tmp/pose-temp-independent-r10; the manifest records exit codes, spec facts, all findings and SHA-256 hashes. No raw repository-wide tool output is embedded.

All baseline scripts and module tests finished before tracked mutations; mutations were sequential and restored byte-for-byte. Intentional leaks were removed. A preview accidentally run during a mutation was rejected and discarded; a later preview against the restored old index was also rejected. The retained final preview follows artifact regeneration on restored implementation paths. Native generated assessment/index files were copied externally and restored. Only the two review-owned files remain modified.

## Native closeout disposition

Bundle preview: `rvb-12ec261992810661`, prepared and unsealed. No attestation. Strict delivery validation, review verify, review-check, close, lint-spec --strict and check --strict were not run because the medium findings prevent the conditional approval closeout. No lifecycle, tasks, trace, roadmaps or implementation-plan edit, commit, push or upstream contribution was made.

## Rules applied during review

- Bugfix review: POSE workflow and pose-review/pose-spec-closeout skills, controlled scope and explicit dispositions.
- Security: temporary paths, permissions, signals, tokens and evidence integrity.
- Backend Go: branch build, vet and full tests, with no Go-source change.
- Documentation style and delivery evidence: distinguish green execution from requirement satisfaction.
- Knowledge governance: searched prior knowledge; no accepted residual risk or handoff.
- Inter-service, frontend and infrastructure contracts: not applicable. Surface/completion tools remain deferred because approval is blocked.

Resolve F2 and F7 before sealing or closing this spec. No production or test correction was made by the reviewer.
