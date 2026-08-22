---
slug: pose-docs-asset-parity
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-package-channel-delivery
priority: 1
components: release
delivers: governance:docs-asset-parity
---

# Spec: A documented download either resolves or fails the release

## 1. Intent

### Goal
Require every `releases/download/...` URL the documentation hands a user to
name an asset the release actually publishes, checked against the real release
after every cut.

### Business value
`docs-site/docs/package-channels.md` offered a copyable
`brew install --formula .../releases/download/vX.Y.Z/pose.rb` across four
releases. `pose.rb` was never uploaded, so the command 404'd for anyone who
tried it. `pose-package-channel-delivery` fixed the upload; nothing stops the
two sides from drifting apart again.

The drift went unnoticed for a specific reason worth naming: both sides were
individually correct. The docs described the intended distribution and the
release published what its workflow was told to publish. Nobody compared them,
because comparing them was nobody's step.

Verified rather than assumed: run against the published v0.20.3 — the last
release before the upload fix — the check passes `checksums.txt` and both
archives and fails on `pose.rb`, reproducing the exact defect it exists to
prevent.

### Constraints
- The check consumes only public release data through `gh`, like the other
  `tests/release/` gates, and shares no producer state.
- Shell is the harness, never the POSE runtime.

### Non-goals
- Verifying that a documented command *works* — that a formula installs, that
  the archive extracts. The clean-host matrix in `package-channels.yml` covers
  installation; this covers existence, which is the failure that shipped.
- Extracting commands from prose. The check reads download URLs, which are
  unambiguous; a natural-language instruction is not.

---

## 2. Requirements

### Functional
- R1: The check shall extract every `releases/download/<version>/<asset>`
  reference from the documentation and expand the version placeholders that
  appear in copyable snippets.
- R2: It shall fail when any extracted asset is absent from the published
  release, naming the asset.
- R3: It shall fail when extraction finds nothing, so a broken extractor
  reports itself instead of passing vacuously.
- R4: It shall run after every published release.

### Non-functional
- One API call for the asset list; the rest is local text processing.

### Security
- Read-only. It downloads nothing and executes nothing from the release.

### Compatibility
- No product change.

---

## 3. Technical Plan

### Affected areas
- `tests/release/docs-asset-parity.sh` — the check.
- `.github/workflows/verify-release.yml` — where it runs.

### Artifacts
- created: .pose/specs/pose-docs-asset-parity/spec.md
- created: tests/release/docs-asset-parity.sh
- modified: .github/workflows/verify-release.yml

### Delivery targets
- governance:docs-asset-parity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- The source list is enumerated by hand (`README.md`,
  `package-channels.md`, `ci.md`). A download URL added to a fourth document is
  invisible to the check — the same shape of gap as the shellcheck file list,
  and recorded rather than solved.
- Placeholder expansion covers `vX.Y.Z`, `${V}` and `$V`, which is what the
  documentation uses today. A new placeholder form would silently produce an
  asset name that never matches, turning a passing check into a failing one —
  noisy rather than silent, which is the right direction.

---

## 4. Tasks

### Planning
- [x] Enumerate the documents that hand a user a download URL
- [x] Confirm the check reproduces the pose.rb defect on a real release

### Implementation
- [x] R1: extract and expand
- [x] R2: fail on a missing asset, by name
- [x] R3: fail on empty extraction
- [x] R4: run it from verify-release.yml

### Validation
- [x] Negative proof against v0.20.3
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
A parity check is only worth its failure path, so it is proven against the
release that actually exhibited the defect rather than against a fixture. The
current release cannot demonstrate the failure — the upload was fixed — so
v0.20.3, the last cut before the fix, is the negative case, and it is a real
published release rather than a constructed one.

### Deterministic checks

#### Security / Contract
- Command: `bash tests/release/docs-asset-parity.sh v0.20.3`
- Scope: documented download URLs against a real published release
- Expected: non-zero exit naming `pose.rb`, with the other three assets passing

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64, `gh` authenticated read.
- Notes: against v0.20.3 the check reports PASS for `checksums.txt`,
  `pose_0.20.3_linux_amd64.tar.gz` and `pose_0.20.3_windows_amd64.zip`, and
  FAIL for `pose.rb`, exiting 1. Shellcheck passes over the new script at
  warning severity.

### Results summary
- Successes: the check reproduces the historical defect and clears the assets
  that were genuinely published
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:docs-asset-parity evidence:integration check:delivery-integration test:tests/release/docs-asset-parity.sh — the check extracts download URLs from the three documents and expands `vX.Y.Z`, `${V}` and `$V` to the release version
- R2 [satisfied] check:docs-asset-parity — run against v0.20.3 it names `pose.rb` and exits non-zero, which is the defect that shipped
- R3 [satisfied] check:docs-asset-parity — an empty extraction exits 1 with a message stating the extractor is broken rather than the docs
- R4 [satisfied] report:.github/workflows/verify-release.yml — the step runs in the workflow that fires on every published release

### Known gaps
- The document list is manual; a download URL in a fourth file is uncovered.
- Existence is not usability: an asset can be published and still be wrong.

---

## 7. Final Report

### Delivered scope
Every documented download URL is checked against the published release after
every cut, and a missing asset fails the verification by name.

### Files and modules changed
- tests/release/docs-asset-parity.sh
- .github/workflows/verify-release.yml

### Validation executed
- Command: `bash tests/release/docs-asset-parity.sh v0.20.3`
- Result: fails on `pose.rb` exactly as the historical defect did; other assets pass

### Residual risks
- A documented URL living outside the three enumerated files is not checked.

### Follow-ups

- [open] The document list is enumerated by hand, which is the same gap shape that left `tests/release/` outside shellcheck for three failed releases. Consider extracting download URLs from all of `docs-site/docs/` and `README.md` by glob instead of by list. (owner:@pose-maintainers crit:low review:2026-11-06)
