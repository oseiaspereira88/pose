---
slug: pose-shellcheck-ci-gate
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-verifier-extension-install-cwd
priority: 1
components: release
delivers: governance:shellcheck-coverage
---

# Spec: Shellcheck covers the release scripts, not just the installer

## 1. Intent

### Goal
Extend the existing CI shellcheck step to every shell script the project ships
or runs, and fail on the severity that catches unbound variables.

### Business value
Three consecutive releases — v0.20.0 through v0.20.2 — failed on defects in
`tests/release/*.sh`. Two of them were `SC2154`, a variable referenced but never
assigned, which aborts the script under `set -u` at the worst possible moment:
after the expensive gates have run and, in one case, after the release was
already published.

The uncomfortable part is that shellcheck was already in CI. It just covered
`install.sh` and `tests/install/run.sh` and stopped there, so the release
scripts — the ones that only execute during a cut, where a defect is most
expensive and least observable — were the ones left out.

Verified rather than assumed: running shellcheck over the v0.20.1 tag's script
reports `independent-verify.sh:96:11: warning: dir is referenced but not
assigned [SC2154]` and exits non-zero. The gate would have caught it before the
tag existed.

### Constraints
- The step must fail the build, not merely report. A warning nobody reads is
  what the previous coverage gap effectively was.

### Non-goals
- Fixing style-level findings (`--severity=style`/`info`). The bar is the class
  of defect that actually shipped.

---

## 2. Requirements

### Functional
- R1: CI shall run shellcheck over every shell script in the repository that is
  shipped or executed, including `tests/release/` and `scripts/`.
- R2: The check shall run at `--severity=warning`, the level at which `SC2154`
  is reported, and a finding shall fail the job.
- R3: The repository shall be clean at that severity when the gate lands.

### Non-functional
- The step stays a single install-and-run; no new tooling to maintain.

### Security
- Shellcheck is static analysis over repository files; it executes nothing.

### Compatibility
- No product change.

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/ci.yml` — the existing Shellcheck step's file list.
- `scripts/release.sh` — one dead assignment, the only finding at this severity.

### Artifacts
- created: .pose/specs/pose-shellcheck-ci-gate/spec.md
- modified: .github/workflows/ci.yml
- modified: scripts/release.sh

### Delivery targets
- governance:shellcheck-coverage module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- A future script added outside the listed paths is silently uncovered again,
  which is exactly how this gap arose. The step enumerates directories rather
  than files to reduce that, but nothing enforces it.

---

## 4. Tasks

### Planning
- [x] Confirm shellcheck reports the defect that shipped, on the real script
- [x] Measure the repository's current findings at that severity

### Implementation
- [x] R1: cover tests/release/, tests/, scripts/ and install.sh
- [x] R2: fail at --severity=warning
- [x] R3: clear the one existing finding

### Validation
- [x] Shellcheck clean over every covered script
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Prove the gate catches the real defect before adopting it: run shellcheck
against the exact script from the v0.20.1 tag and confirm it reports SC2154 and
exits non-zero. Then confirm the current tree is clean at that severity, so the
gate lands green rather than pre-broken.

### Deterministic checks

#### Security / Contract
- Command: `shellcheck --severity=warning install.sh scripts/*.sh tests/*.sh tests/*/*.sh`
- Scope: every shipped or executed shell script
- Expected: no findings

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64; shellcheck via the koalaman/shellcheck:stable image,
  since it is not installed on this host.
- Notes: against `v0.20.1:tests/release/independent-verify.sh` the check reports
  `96:11: warning: dir is referenced but not assigned [SC2154]` and exits 1. The
  current tree reports one finding, `SC2034` for a dead `VERSION_NUM` in
  scripts/release.sh, removed here.

### Results summary
- Successes: gate proven against the real defect; tree clean at warning severity
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:shellcheck-coverage evidence:integration check:delivery-integration report:.github/workflows/ci.yml — the step now covers install.sh, scripts/, tests/ and tests/*/, where it previously covered two files
- R2 [satisfied] check:shellcheck — the step runs at --severity=warning and its non-zero exit fails the job; verified by running that severity against the broken v0.20.1 script
- R3 [satisfied] check:shellcheck — the only finding at that severity, a dead assignment in scripts/release.sh, was removed; the tree reports none

### Known gaps
- Style and info level findings are not enforced, deliberately.
- Nothing prevents a future script from being added outside the covered paths.

---

## 7. Final Report

### Delivered scope
The CI shellcheck step covers every shipped or executed shell script instead of
two, at the severity that reports unbound variables, and fails the build on a
finding. The one pre-existing finding was cleared so the gate lands green.

### Files and modules changed
- .github/workflows/ci.yml
- scripts/release.sh

### Validation executed
- Command: shellcheck at --severity=warning over every covered script
- Result: no findings; the same command against the v0.20.1 script reports SC2154

### Residual risks
- Shellcheck would not have caught the third failure in the sequence — the
  missing `cd` before `pose extension install` is valid shell. Static analysis
  closes one class of defect here, not the category.

### Follow-ups

- [open] The missing-`cd` defect was valid shell and invisible to static analysis. The durable answer is reproducing a gate's exact command locally before cutting, which is process rather than tooling — consider whether the release workflow can execute the verification legs against a snapshot before a tag exists. (owner:@pose-maintainers crit:medium review:2026-10-02)
