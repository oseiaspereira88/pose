# Independent review, round 11: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `70a330c5670c6aaa2077d524d7ca227b64088f24`. Exhaustive sweep completed before deciding: **F2 and F7 remain medium findings**. No implementation correction, seal, attestation, closeout, commit or push was made.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, maintainer authorized. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is an engine-required declaration until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied, not independently authenticated.

Reviewed the full branch diff and round-11 correction, spec, AGENTS.md, POSE review/closeout instructions, applicable rules and pose-review/pose-spec-closeout skills. Built and used the requested branch CLI. Requirements and Technical Plan were not edited. The round-10 record is retained in commit 70a330c5; these reviewer-owned files now describe round 11.

## Complete findings

### F2 — Medium: logical-line parsing and state checks still approve leaking scripts

Locations: `tests/hygiene/temp-cleanup.sh:138` and `:146` (continuation spaces), `:132` and `:140` (heredoc parsing/consumption), `:198` and `:210` (missing allocations and exit cutoff), `:235` (helper-init context), `:206`, `:219`, `:229` and `:262` (state/option checks). Requirement: **R3**.

The exact scan accepts twenty new programs that independently leak. Every one also makes the **complete hygiene gate return 0** when temporarily substituted into tracked tests/import/migration-guides.sh. Seventeen pass ShellCheck at the CI warning severity. The other three have separate ShellCheck diagnostics and are recorded as scan counterexamples, without claiming they pass CI as a whole.

The complete-gate controls do not execute the substituted migration script: they demonstrate static admission. Separate executions prove each leak. The gate's own TMPDIR is empty in all twenty runs. One counterexample deliberately unexports TMPDIR; its caller directory is empty too, but a directory survives outside it, so an empty-parent assertion alone would miss that escape.

**1. Continuation changes command words rather than preserving shell tokens.** Backslash-newline is removed by Bash, including within a word; the scanner inserts a space. `mk\` followed by `temp -d` executes mktemp, leaks and is approved as containing no mktemp. A helper-using script with `tr\` followed by `ap ':' EXIT` silently overrides cleanup while the scanner sees two different words. An ordinary backslash-escaped character in mktemp produces the same missed-command class, with a separate ShellCheck diagnostic.

**2. An init line is not necessarily an unconditional top-level command.** After a physical line ending in false && or true ||, pose_test_tmp_init is skipped. After printf followed by a pipe, init runs in the pipeline's subshell, whose trap does not protect the parent. The parser emits the init as a separate top-level line and accepts all three. For example, this warning-clean program returns 0 and leaves w:

```bash
set -euo pipefail
. /absolute/path/to/tests/lib/tempdir.sh
false &&
pose_test_tmp_init
w="$(mktemp -d)"
: "$w"
```

**3. Valid heredoc forms can hide code or fabricate an executed init.** With an escaped delimiter (`<<\EOF`) or numeric delimiter (`<<123`), the parser recognizes no delimiter and scans the body as executable source/init, although Bash only prints it. With a concatenated quoted delimiter (`<<E"OF"`), the parser records E, consumes through EOF without finding it, and silently loses the allocation; Bash correctly terminates at EOF. The latter is valid for Bash but receives a separate ShellCheck parse diagnostic. A continued heredoc header is consumed too early: `cat <<EOF` followed by backslash-newline, `; trap ':' EXIT`, body, EOF hides a real trap override from the scan. Escaped/numeric/header-continuation cases pass warning-level ShellCheck. Unrecognized or unterminated lexer delimiters must be rejected instead of accepted as no allocation.

**4. A syntactic exit command can fail without ending the script.** A failed redirection on `exit > /nonexistent/directory/file || true` prevents execution of the builtin; the next mktemp runs. The scanner cuts at exit and returns success. This passes warning-level ShellCheck. `exit 0 extra || true` similarly returns an error without exiting and leaks, although ShellCheck separately flags its arguments.

**5. Path-variable attributes can defeat the helper.** `typeset +x TMPDIR` removes the export after init, so the next external mktemp allocates outside the owned root and survives. `declare -i POSE_TEST_TMP` or `declare -i TMPDIR` before init makes the pathname assignment fail as arithmetic before cleanup is installed; the observed scripts return 0 and leave both allocations. `declare -u POSE_TEST_TMP` uppercases the path, so later allocation/cleanup target a different name. All four are accepted and warning-clean with explicit variable use. There is no eval or hidden sourced implementation in these cases.

**6. The noexec/permission filters still inspect spelling rather than effective options.** `set -e -n` disables the EXIT handler but is accepted because n is not in the first option word. `chmod "0000" "$w/a"` and `chmod -- 0000 "$w/a"` are accepted, yet make the own-trap rm fail on a/b/f. All three are warning-clean and pass the complete gate.

**7. A literal assignment is not guaranteed safe before the trap.** With `declare -i label` before allocation and set -u, the otherwise allowed gap assignment `label="POSE_INDEPENDENT_UNSET_ARITH"` performs an arithmetic lookup of an unset variable and exits before trap registration. It leaks and is warning-clean. A division-by-zero sibling probe did reach cleanup and is excluded from findings; the unset-variable case actually leaks.

Every accepted leaking case:

| Case | Script exit | Observed residue | ShellCheck warning exit |
| --- | --- | --- | --- |
| `mktemp-word-continuation` | 0 | caller root nonempty | 0 |
| `mktemp-word-escape` | 0 | caller root nonempty | 1 |
| `helper-trap-word-continuation` | 0 | caller root nonempty | 0 |
| `helper-init-after-and` | 0 | caller root nonempty | 0 |
| `helper-init-after-or` | 0 | caller root nonempty | 0 |
| `helper-init-in-pipeline` | 0 | caller root nonempty | 0 |
| `helper-unexport-typeset` | 0 | outside caller root | 0 |
| `own-trap-stacked-noexec` | 0 | caller root nonempty | 0 |
| `own-trap-quoted-chmod` | 1 | caller root nonempty | 0 |
| `own-trap-chmod-end-options` | 1 | caller root nonempty | 0 |
| `helper-trap-hidden-before-heredoc-body` | 0 | caller root nonempty | 0 |
| `escaped-heredoc-delimiter` | 0 | caller root nonempty | 0 |
| `numeric-heredoc-delimiter` | 0 | caller root nonempty | 0 |
| `concatenated-heredoc-delimiter` | 0 | caller root nonempty | 1 |
| `exit-redirection-fails` | 0 | caller root nonempty | 0 |
| `exit-too-many-arguments` | 0 | caller root nonempty | 1 |
| `integer-helper-root` | 0 | caller root nonempty | 0 |
| `integer-helper-tmpdir` | 0 | caller root nonempty | 0 |
| `uppercase-helper-root` | 1 | caller root nonempty | 0 |
| `integer-unset-gap` | 1 | caller root nonempty | 0 |

Expected action: normalize tokens and execution context consistently or reject unsupported constructs before deciding there is no allocation. Do not infer unconditional init from a physical/logical line alone or unconditional termination from the exit prefix. Enforce exported mutable path variables and inspect complete option/attribute forms. Require the twenty tracked-path controls to reject or demonstrably clean up. These are visible shell syntax and state changes, rather than the declared eval/source opacity exclusions; no proof of arbitrary shell is being required.

### F7 — Medium: CDPATH contaminates the absolute-root capture

Location: `tests/lib/tempdir.sh:15`, consumed at `:19`, `:30–31`. Requirements: **R1**, **R2**, and the caller-TMPDIR constraint.

The relative-TMPDIR correction succeeds with CDPATH unset. With exported CDPATH=. and TMPDIR=tmp, the new cd command prints its destination, and pwd -P prints it again. The command substitution stores two absolute paths separated by a newline. That value is neither a valid exported TMPDIR nor the actual removal target.

Minimal scenario: start in a disposable directory containing tmp; source the real helper in a Bash process with CDPATH=. and TMPDIR=tmp; call init, cd /, true. It exits 0 and leaves its root. The captured OWNED value independently shows the duplicate newline-separated pathname. The three unchanged current scripts, invoked by absolute script paths from that directory, each exit 1 on their next mktemp and leave the helper root. Install uses absolute --binary and --previous 6.2.0, and fails before network access. The equivalent four runs with CDPATH unset return 0 and leave no entries.

Expected action: make the absolute-path capture independent of caller CDPATH and guarantee one pathname. Revalidate the helper and all three real scripts with relative TMPDIR, exported CDPATH and directory changes. This remains F7 as a partial correction, rather than a new identifier for the same caller-relative-path contract.

## Corrections and requirement judgment

All eleven round-10 cases are now rejected; 67 earlier external corpus entries are also rejected. The built-in 91 negatives, 5 positives and 3 tracked mutations pass. F1, F3, F4, F5 and F6 remain resolved in their original reviewed scope. Twelve nested permission/failure cases clean up, the exact shipped setup reaches markers and exits 0/1/1/1, helper INT/TERM exit 130/143 and clean up, and removing INT handling or restoring old chmod makes the gate fail. The gate itself exits 130/143 on signals and leaves no residue on those or negative failure controls. Composition docs name pose_test_tmp_remove.

The demo still allocates one root: a shim that would fail a second mktemp observes only one call, --verify succeeds with an empty TMPDIR, and GOTOOLCHAIN=invalid exits 1 with an empty TMPDIR. Assertions and source/init/cd/argument handling of the three targeted scripts and demo remain unchanged except for cleanup setup and the reviewed single-root sequencing.

R1 and R2 pass ordinary absolute and relative TMPDIR runs, but F7 remains an observed failure. R3 remains unsatisfied by F2. Restoring the origin/main scripts makes the static check fail; the old quickstart returns 0 and leaves a directory, failing the empty-parent condition. The no-helper control works. Empty caller TMPDIR alone is insufficient for F2's unexport scenario, whose escaped directory was separately observed and removed.

D1 remains proportionate: one owned root captures child Go work and curl/git fixtures without changing assertions. The absolute-path implementation still needs the CDPATH correction. Ordinary child tools do not depend on retaining the caller's original TMPDIR. SIGKILL independently exits -9 and leaves a root, an unavoidable trap limitation requiring disclosure before closeout; Final Report is Pending. Eval/source opacity is already declared; observed F2/F7 defects are not accepted residual risks.

All 24 tracked *.sh files fall under tests/, scripts/, examples/ or install.sh; Git pathspecs include descendants. Helper/hygiene are statically exempt and dynamically tested. No inter-module/API/MCP contract changed.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over CI globs, the shipped hygiene gate and TestLocalVerifyCoversCurrentCIGates pass. Quickstart, migration, demo and quickstart --binary runs leave absolute TMPDIR empty. The initial install journey received HTTP 403 while listing releases and still cleaned up; an authenticated retry with GH_TOKEN limited to that command passes and cleans up. No secret was printed. Install also passes with --binary/--previous 6.2.0 from another directory. All three scripts pass relative TMPDIR with CDPATH unset.

CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised gate. Artifact-check exits 0: all 12 claims are observed among 14 paths, with the two review artifacts as extras and no finding naming this spec. Tech-debt reports zero markers/uncovered items; recurrence flags zero keys. assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Raw outputs and executable probes are outside the repository in /tmp/pose-temp-independent-r11; the manifest contains exit codes, spec facts, all findings and SHA-256 hashes. Full native output is not embedded.

Mutations ran sequentially, without overlap with real-script or Go-suite runs, and were restored byte-for-byte. The authenticated journey retry ran between completed mutation batches on restored source. All intentional leftovers, including the escaped directory, were observed and removed. Native generated assessment/index files were retained externally and restored; only the two reviewer-owned files remain modified.

## Native closeout disposition

Prepared preview: `rvb-10c5f3bce2e1ed87`, unsealed. No attestation. Strict delivery validation, review verify, review-check, close, lint-spec --strict and check --strict were not run because medium findings prevent the conditional approval closeout. Tasks, trace, lifecycle, implementation plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- Bugfix review: POSE review workflow and pose-review/pose-spec-closeout skills; scope, contracts, adversarial gate value and explicit dispositions.
- Security: paths, export state, permissions, signals, token handling and evidence integrity.
- Backend Go: branch build, vet and full tests, without Go-source modification.
- Documentation style and delivery evidence: preserve Requirements/Technical Plan and distinguish green checks from requirement satisfaction.
- Knowledge governance: prior knowledge consulted; no accepted residual risk or handoff.
- Frontend, infrastructure and inter-service contract rules: not applicable. Surface/completion tools are deferred because approval is blocked.

Resolve F2 and F7 before sealing or closing. No production or test correction was made by the reviewer.
