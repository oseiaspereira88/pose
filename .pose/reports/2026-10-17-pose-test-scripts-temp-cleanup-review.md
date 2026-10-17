# Independent review, round 8: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `091869f922987655b1ef126fde63acb55137e945`. Exhaustive sweep completed before deciding: F2 remains a medium finding, with five accepted executable leaks. No separate new finding was identified. No persistent implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. Individual implementer/session identity is maintainer-supplied and not independently authenticated; the prefix does not establish signed execution provenance.

Reviewed the full branch diff, correction diff, spec, POSE review/closeout instructions, skills and applicable rules. Built and used the requested branch CLI. Requirements and Technical Plan were not edited by this reviewer. Previous-round records remain in commit 091869f9; these owned artifacts now report round 8.

## Complete findings

### F2 — Medium: future bindings and unsafe assignment contexts still pass

Location: `tests/hygiene/temp-cleanup.sh:226`, also lines 46–47 and 232–237. Requirement: **R3**.

The five prior cases are corrected; all 54 retained external synthetic leak cases are now rejected. The 73 built-in negatives, 5 positives and three tracked-copy mutations also pass. The new sweep confirms five further accepted executable leaks, all clean under **ShellCheck --severity=warning**.

**Future variables are treated as already initialized.** `defined` starts with every allocated variable, including allocations that have not run yet. This script is accepted:

```bash
set -euo pipefail
a="$(mktemp -d)"
copy="$b"
b="$(mktemp -d)"
trap 'rm -rf "$a" "$b"' EXIT
: "$copy"
```

At runtime b is unset at the copy. The script exits 1 before trap registration, leaving a. Temporarily substituting it into the tracked migration-guides path makes the **complete hygiene gate return 0**. The claimed “already set” check therefore does not establish execution-order initialization.

**Readonly state makes an accepted literal assignment or allocation fail.** The sweep reproduces three contexts: writing a literal to a previously readonly label in the allocation-to-trap gap; assigning a mktemp result to a readonly target (the substitution creates the directory before binding fails); and calling the helper after declaring TMPDIR readonly. The last case creates the helper root and aborts when exporting the replacement TMPDIR, before its traps are registered. All three exit 1 and leave the caller TMPDIR nonempty. They need rejection as unsupported contexts or safe handling before allocation; a syntactically plain assignment alone does not prove that it cannot fail.

**let can have an actual heredoc redirection.** The command-wide shortcut at line 46 returns before examining redirections. This valid script is accepted:

```bash
let 1 <<'DOC'
source tests/lib/tempdir.sh
pose_test_tmp_init
DOC
w="$(mktemp -d)"
: "$w"
```

The source/init lines are heredoc data and never execute. The scanner mistakes them for top-level helper initialization; runtime returns 0 and leaves w. The executed fixture uses the absolute path of the shipped helper inside that data. Its tracked-path substitution also makes the **complete hygiene gate return 0**. No sourced file actually runs and no eval is involved, so the declared arbitrary-shell residual does not account for this acceptance.

All five finding cases are recorded together:

| Accepted case | Exit | Result |
| --- | --- | --- |
| Reference to a future mktemp variable | 1 | Earlier root remains |
| Literal assignment to a readonly label before trap registration | 1 | Root remains |
| mktemp substitution bound to a readonly target | 1 | Created root remains |
| Helper initialization with readonly TMPDIR | 1 | Helper root remains |
| Real heredoc on let contains fake helper initialization | 0 | Allocation remains |

Three correct controls (two-root trap, defined-variable path assignment, helper) pass and clean up. An integer-assignment experiment that leaves no residue is excluded from findings. All five leaks pass ShellCheck at CI severity, so that gate does not compensate.

Expected action: track definite initialization in source/execution order; account for or reject unsupported readonly metadata before allowing allocation and gap assignments; parse real heredoc redirections on let rather than skipping the entire command. Require all five cases and both complete-gate controls to fail. Preserve the explicit boundary for arbitrary eval/sourced-file behavior without treating these visible constructs as proof of cleanup.

## Previous findings and requirement judgment

F1, F3, F4, F5 and F6 remain resolved in their reviewed scope. Twelve helper cases cover nested 0500, 0400 and 0000 entries across success, assertion, set-e and unset-variable exits, all leaving TMPDIR empty. The exact shipped body reaches its setup marker and returns 0/1/1/1. Helper INT/TERM return 130/143 and clean up; removing INT handling makes the gate fail and clean up. Composition instructions name pose_test_tmp_remove.

The hygiene command returns INT 130 and TERM 143 at the former permission-control failure point, leaving no sandbox. The old-chmod mutation also fails with exit 1 and cleans up. External DEBUG hooks select only signal timing. These checks use mutable helper state; F2 separately identifies admission of readonly contexts.

The demo passes normal --verify execution and the GOTOOLCHAIN=invalid build failure returns 1 with no residue. Its adjacent allocation/trap correction and artifact claim remain valid; its assertions are unchanged.

R1 helper/hygiene behavior passes the original mutable-state matrix. R2 passes for all three real scripts with independent TMPDIR roots. Migration and install run from another directory; quickstart additionally passes there with an absolute --binary. Install also passes with --binary and --previous 6.2.0. GitHub works anonymously; no token was needed. Real-script and module-suite runs completed before temporary mutations.

R3 remains unsatisfied (F2). Restoring the three scripts from origin/main makes the scan fail. The old quickstart returns 0 but leaves a directory, failing the nonempty-directory condition. The no-helper control works. These prove rejection of the original defects while the five cases establish the remaining gap.

The full diff preserves the three real-script assertions and source/init/cd/argument behavior. TMPDIR captures ordinary Go work directories and allocated curl/git fixtures, with no dependency on retaining the caller's original value. All tracked *.sh currently lie in tests/, scripts/, examples/ or install.sh; Git pathspecs cover descendants. Helper/hygiene are statically exempt and dynamically exercised.

D1 remains proportionate: one owned root captures descendant outputs without maintaining a list. SIGKILL independently leaves its root, an unavoidable trap limitation to disclose at closeout; Final Report is Pending. F2 is an observed defect, not accepted residual risk.

## Checks and compact evidence

Branch CLI build, go vet ./..., go test ./... -count=1, ShellCheck 0.11.0 --severity=warning over the CI glob set, hygiene and TestLocalVerifyCoversCurrentCIGates pass. Each of the five leaking fixtures separately passes the same ShellCheck severity. CI, scripts/verify.sh and required matrix check test-scripts-remove-their-temp-files-integration invoke the exercised hygiene command.

Artifact-check exits 0: all twelve declared claims are observed. Fourteen observed paths include the two prior review artifacts, with no native finding naming this spec. Implementation commits carry POSE-Spec attribution. Tech-debt assessment finds zero markers/uncovered items; recurrence-check flags zero keys. No MCP/REST/Protobuf/Kafka contract changed; assess integrate is not applicable.

Compact evidence: [test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). Full output and executable probes remain outside the repository at /tmp/pose-temp-independent-r8; the manifest stores exit codes, facts, all cases and SHA-256 hashes. Mutations ran sequentially and were restored byte-for-byte. All deliberate leaks were observed then removed. Native assessment/index side effects were copied outside the repository and restored to their initial tracked state.

## Native closeout disposition

Prepared preview: `rvb-e929e1645b47e9d5`, unsealed. No sealed bundle or attestation was created. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because the medium finding prevents entering the conditional approval closeout. Tasks, trace, plan and roadmaps remain unchanged. No commit, push or upstream contribution was made.

## Rules applied during review

- Type: bugfix with CI/validation integration; .pose/workflows/review.md and pose-review/pose-spec-closeout skills.
- Security: temporary paths, permissions, signals, token handling and evidence integrity.
- Documentation style: spec, cleanup instructions and report accuracy.
- Backend Go: selected by the effective plan; branch build, vet and full tests exercised, with no Go-source change.
- Delivery evidence: distinguish successful checks from requirement satisfaction.
- Knowledge governance: consulted; no accepted residual risk or handoff.
- Frontend, infrastructure and inter-service contract rules: not applicable.

## Next action

Resolve F2 for the claimed supported forms and make the five negative cases and both complete-gate controls reject them before requesting another independent review or sealing delivery.
