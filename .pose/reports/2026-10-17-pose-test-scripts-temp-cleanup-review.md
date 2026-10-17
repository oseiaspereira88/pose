# Independent review: test-script temporary cleanup

Decision: **changes requested**. PR #192, branch `impl/pose-test-temp-cleanup`, reviewed at `fbe46f0e8d49c80a9b62d8cc0f01c741530e5455`. Complete findings follow; no implementation correction, seal, attestation, closeout, commit or push was performed.

## Scope and identity judgment

Reviewer: `agent:independent-gpt-6.1-sol-review`, authorized by the maintainer. This is a Codex/OpenAI execution reviewing Claude/Anthropic work, hence a different vendor. The `agent:independent-` prefix is a declaration required by the engine until `pose-delegated-review-run-provenance` replaces it. Identity is declared; pose-dist attestations are unsigned. The maintainer identifies Claude Opus 5.5 session `296d1fcb-e989-40ae-9362-cf3514db8b26`; the three implementation commits instead carry Claude Sonnet 5.5 trailers and a separate claude.ai session URL. Neither individual-session claim was independently authenticated. This does not establish signed execution provenance.

Read the complete branch diff, spec, operating review and closeout instructions, selected skills and applicable rules. Three implementation commits carry the contiguous POSE-Spec trailer; native artifact-check resolves their range and matches all 11 claims to 11 observed paths, with no finding for this spec. The initial prepared bundle could not observe the attribution before artifact-check refreshed the index; preparation after that succeeded. No policy bypass was attempted.

## Findings

### F1 — medium: `tests/lib/tempdir.sh:26`

**Scenario:** Create a nested directory containing a file, then chmod 0400 or 0000 that directory before normal exit. chmod -R u+w does not restore directory traversal. rm fails with Permission denied, exit is 1 and TMPDIR remains nonempty. Mode 0500 is cleaned successfully.

**Affected contract:** R1. **Expected action:** Restore the directory permissions needed for traversal and removal, including nested inaccessible directories, and cover these permission cases.

### F2 — medium: `tests/hygiene/temp-cleanup.sh:28`

**Scenario:** The exact static scanner returns 0 for an already tracked script containing mktemp plus each of: cleanup() { :; }; trap cleanup EXIT, a commented rm trap, a TERM trap with EXIT in a comment, an EXIT trap deleting an unrelated path, or only a comment ending in pose_test_tmp_init. None guarantees removal of its temporary directory.

**Affected contract:** R3. **Expected action:** Make acceptance establish an executable cleanup mechanism covering the temporary paths. Add negative cases for comments, no-op cleanup, wrong signals and fake initialization; do not describe lexical presence as proof of removal.

### F3 — medium: `tests/hygiene/temp-cleanup.sh:54`

**Scenario:** The asynchronous bash launch inherits SIGINT ignored. Reproducing the exact launch prints trap -- '' SIGINT; the process remains alive after INT, waits for its sleep, exits 0 and leaves TMPDIR empty. The hygiene gate reports INT covered even though cleanup occurred on ordinary completion. Its wait status is discarded.

**Affected contract:** R1 validation; R3 meaningful gate. **Expected action:** Launch the INT probe with SIGINT enabled, assert signal-specific termination and prompt completion, and ensure a missing INT handler causes the probe to fail.

### F4 — low: `tests/lib/tempdir.sh:9`

**Scenario:** The helper tells future callers to compose a custom trap with pose_test_tmp_on_exit, but defines only pose_test_tmp_remove. Following the instruction invokes a nonexistent function after replacing the helper EXIT trap and can leave the directory.

**Affected contract:** D1 trap composition documentation. **Expected action:** Name the actual cleanup function in the composition instructions and show a valid composition.

## Requirement and decision judgment

R1 is not satisfied across the independently tested permission cases (F1). With ordinary readable/traversable read-only directories, independent helper probes returned success 0, assertion 1, set-e error 1, unset variable 1, INT 130 and TERM 143, all leaving their parent TMPDIR empty. A direct SIGKILL probe left its directory. SIGKILL cannot run a trap; state this residual boundary before closeout rather than treating D1's unqualified “a way to end” falsifier as achievable for all termination modes. F3 means the shipped INT test does not establish the behavior it claims.

R2 passed: all three real scripts returned 0 and left independent TMPDIR roots empty. Migration and install journeys ran from another directory; quickstart also passed from another directory with an absolute --binary. The install journey additionally passed with explicit --binary and --previous 6.2.0. The API worked anonymously; no token was needed. Source/init precede the first mktemp, and subsequent cd/argument behavior is preserved. The line-by-line diff changes no assertion: the sole removed statement is the quickstart's trailing successful-path rm, superseded by EXIT cleanup. Exported TMPDIR keeps Go work directories and explicitly allocated curl/git fixtures under the owned root; none of these scripts depends on retaining the caller's TMPDIR value. User caches outside it remain outside the declared scope.

R3 is not satisfied (F2). Restoring all three scripts from origin/main made the gate fail at migration-guides. Running the old quickstart separately returned 0 but left a tmp.* directory, which fails the gate's nonempty-directory condition. The shipped no-helper control also demonstrated a leak. The production static section was executed verbatim against five synthetic replacements of an already tracked script, restoring original bytes afterward; all five leaky variants were accepted. No shell parsing coverage or adversarial corpus prevents these form-only passes. All tracked *.sh currently fall within the declared directories or install.sh; git ls-files pathspecs include nested scripts. Excluding other directories is consistent with R3 and creates no current tracked-shell gap.

D1 is proportionate: one owned TMPDIR root also captures descendant command artifacts and avoids maintaining a list of mktemp paths. It needs permission handling corrected and truthful enforcement evidence before acceptance. Future scripts that only resemble cleanup are an observed gate defect, not a risk accepted by this review. The spec's Residual risks and Final Report remain Pending; no final residual-risk claim has been approved.

## Validation and evidence

Branch CLI build, go vet ./..., go test ./... -count=1, the hygiene command, the explicit TestLocalVerifyCoversCurrentCIGates run and ShellCheck 0.11.0 --severity=warning over the CI glob set passed. The CI step, local verifier and required matrix check test-scripts-remove-their-temp-files-integration invoke the same exercised hygiene command. Tech-debt assessment found zero markers and zero uncovered items; recurrence-check flagged zero keys. No public inter-module contract changed, so assess integrate is not applicable.

The compact evidence manifest is [.pose/results/test-scripts-temp-cleanup-independent-review.json](../results/test-scripts-temp-cleanup-independent-review.json). It records exit codes, spec facts, all findings and SHA-256 digests. Full command output and probe sources remain outside the repository at /tmp/pose-temp-independent. Assessment/index side effects were copied there and restored to the original tracked state. All temporary test mutations were restored; deliberately leaked fixtures were observed and then removed.

## Native closeout disposition

Prepared preview: rvb-b182ef8a8e275fc2 (unsealed). There is no sealed bundle ID and no attestation ID. Review verify, close, lint-spec --strict, check --strict and strict delivery validation were not run because the conditional approval closeout was not entered. No plan or roadmap was marked delivered. No commit was created or pushed, and no upstream contribution was staged or submitted.

## Rules applied during review

- Change type: bugfix, with CI and validation integration.
- Workflow: .pose/workflows/review.md; skills pose-review and pose-spec-closeout.
- Security: selected for token handling, temporary paths and evidence integrity.
- Documentation style: selected for spec, composition instructions and review report.
- Backend Go: selected by the effective review plan; Go build, vet and tests exercised, with no Go source changes.
- Delivery evidence: selected to keep passing commands distinct from requirement satisfaction.
- Knowledge governance: consulted for applicable handoff obligations; no residual risk was accepted and no knowledge artifact was created.
- Frontend, infrastructure and cross-service contract rules: not applicable to this shell hygiene diff.

## Next action

Resolve F1–F3 and correct F4, then request a fresh independent review against the changed subject. Keep Requirements and Technical Plan intact unless a separately authorized scope decision changes them. Re-run adversarial negatives before any seal or closeout.
