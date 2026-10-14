# Security validation release API authentication — 2026-10-14

## Observed failure and root cause

[Security #742](https://github.com/oseiaspereira88/pose/actions/runs/38504224761),
[job validation-findings](https://github.com/oseiaspereira88/pose/actions/runs/38504224761/job/115657893577),
head `2595a234bda0f046fb558d2a0350d6a1a93fcf24` of PR #171:
`install-and-upgrade-journeys-integration` failed while listing published releases.
The log shows `curl: (22) The requested URL returned error: 403`; journey 1 and 1b
passed before the API call. It is not a failure of the newly authored draft specs.

The caller in `.github/workflows/security.yml` supplies only the history promise.
The registered check sets POSE_JOURNEYS=1; validate inherits os.Environ and the Go
test inherits it into Bash. The script already uses GITHUB_TOKEN/GH_TOKEN when set,
but this step supplies neither, so the listing is anonymous. The direct journey
step in ci.yml supplies github.token and [CI #740](https://github.com/oseiaspereira88/pose/actions/runs/38504224767)
passed on the same head. Shared anonymous runner rate-limit exhaustion is the
likely cause of the 403, consistent with the prior
[spec](../specs/2026-10-10-pose-ci-avoids-anonymous-rate-limits.md); the failed log
contains no response headers/body proving that precise GitHub reason.

## Surgical correction

Pass GITHUB_TOKEN from github.token in the existing Security validation step,
retaining its history promise, validation command, required journey and job
permissions. No new credential, permission, runtime behavior, dependency, policy
or consolidation-unit implementation. Keep PR #171 documentary and publish this
configuration correction separately against main. Reuse the existing journey and
workflow tests; no duplicate test for a reversible environment binding.

## Verification

Local Go 1.26.0 workflow-contract attempt failed before test execution with
`go: error obtaining buildID for go tool compile: signal: segmentation fault`.
This confirms the previously recorded environment limitation; it is not a test
verdict and was not repaired by this correction. GitHub uses Go 1.26.9.

Record YAML equivalence, POSE gates and corrected GitHub result below. Do not
claim the corrected network journey passed until its runner provides evidence.
Spec remains in-progress through review/native closeout.

### Local results

- YAML before/after parsed successfully; removing only the added token binding from the selected step makes the complete documents equal. Command, permissions, pins, other steps and history promise are unchanged.
- `pose lint-spec pose-security-validation-release-api-auth --strict`: exit 0; optional Tasks warning.
- `pose artifact-check --spec pose-security-validation-release-api-auth --from 06a5444d6a51d5e5d66a3715e02f71067db38a6c --to HEAD --strict --json`: exit 0, four claims/four observed paths, zero errors and 606 orphan warnings for other paths outside the filtered spec.
- `pose check --strict`: exit 0, `Result: SUCCESS — (strict mode) with 30 warning(s).`; historical changelog, freshness and instance stamp warnings retained.
- `git diff --check`: passed. Local POSE gates use the existing 7.4.1 binary, without claiming a source build or Go test verdict.

These checks validate the configuration delta and its provenance. The existing journey on the corrected GitHub runner is pending at PR creation; its workflow URL and verdict must be consulted before asserting network verification. The failed original log remains evidence of the incident, not of corrected behavior.

| Requirement | Evidence at PR creation |
|---|---|
| R1 | Token binding present in the selected step, inherited by validate/test/script; runtime verification pending. |
| R2 | Whole-document YAML equivalence after removing that binding; no change to permissions, commands, pins or history promise. |
| R3 | Failed job and direct CI control linked above; native structural gates passed with disclosed warnings; corrected journey pending GitHub. |
