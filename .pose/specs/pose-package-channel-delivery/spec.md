---
slug: pose-package-channel-delivery
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-package-manager-distribution
priority: 1
components: release
delivers: governance:package-channel-delivery
---

# Spec: The package-manager install path documented is the one that exists

## 1. Intent

### Goal
Publish the Homebrew formula and WinGet manifests as release assets, and give
the clean-host verification a trigger that fires and a command that runs.

### Business value
`docs-site/docs/package-channels.md` hands a user a copyable command —
`brew install --formula .../releases/download/vX.Y.Z/pose.rb` — and states the
channel is "exercised on every tagged release by the clean-host matrix". Neither
holds: `pose.rb` is not among the release assets, and `package-channels.yml` has
zero runs across every release the project has cut.

Two independent defects produced that. The workflow triggers on
`release: published`, which never fires for a release created by the workflow's
own `GITHUB_TOKEN`. And when dispatched by hand it dies in 26 seconds on both
runners: `go run ./pose-mcp/cmd/pose` executes from the repository root, which
is not a Go module — the correct form sits a few lines away in `release.yml`.

While the project is private this is debt. On public release it is the first
thing a user tries, and the documentation asserts it works.

### Constraints
- The manifests are derived from `checksums.txt`, which goreleaser produces, so
  they can only be uploaded after it runs.

### Non-goals
- Submitting to `winget-pkgs` upstream. That is a manual, separately tracked
  step.
- Owning a Homebrew tap.

---

## 2. Requirements

### Functional
- R1: `pose.rb` and the three WinGet manifest files shall be published as
  release assets, so the documented URL resolves.
- R2: The clean-host verification shall run automatically after a release,
  using a trigger that actually fires.
- R3: The manifest generation shall invoke the Go module where it lives.

### Non-functional
- The upload adds one step to the release job and no new tooling.

### Security
- The upload uses the job's existing token and targets only the tag being cut.

### Compatibility
- Additive: four new release assets, no existing asset changed.

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/release.yml` — publish the manifests as assets.
- `.github/workflows/package-channels.yml` — trigger and command.

### Artifacts
- created: .pose/specs/pose-package-channel-delivery/spec.md
- modified: .github/workflows/release.yml
- modified: .github/workflows/package-channels.yml

### Delivery targets
- governance:package-channel-delivery module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- `workflow_run` fires for every completed Release run, including failed ones;
  the job guards on conclusion and a `v*` head branch, mirroring
  verify-release.yml.

---

## 4. Tasks

### Planning
- [x] Confirm pose.rb is absent from the published assets
- [x] Confirm the workflow has never run and why

### Implementation
- [x] R1: upload the four manifest files after goreleaser
- [x] R2: add the workflow_run trigger with a conclusion guard
- [x] R3: run the generator from pose-mcp/

### Validation
- [x] Generator produces the four files locally
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Run the generator locally against the published v0.20.3 checksums to prove the
four files are produced and named as the documentation expects. The upload and
the clean-host matrix are only provable on a real cut, which is the next one.

### Deterministic checks

#### Security / Contract
- Command: `pose release-package-manifests --version 0.20.3 --checksums checksums.txt --out manifests`
- Scope: Homebrew formula and WinGet manifest set
- Expected: `homebrew/pose.rb` plus three `winget/Harne8.Pose*.yaml`

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.20.3-dev.
- Notes: `gh release view v0.20.3` lists no `pose.rb`; `gh run list --workflow
  package-channels.yml` returns nothing across the project's history; the one
  manual dispatch (run 31199572548) failed in 26s on macOS and Windows with
  "cannot find main module".

### Results summary
- Successes: generator verified locally; trigger and command corrected
- Failures: none
- Warnings: the upload and the clean-host matrix run for the first time on the
  next cut

### Requirement trace
- R1 [satisfied] governance:package-channel-delivery evidence:integration check:delivery-integration report:.github/workflows/release.yml — the release job uploads pose.rb and the three WinGet files to the tag being cut, so the documented URL resolves
- R2 [satisfied] report:.github/workflows/package-channels.yml — the workflow_run trigger fires where release:published never did, guarded on a successful Release run and a v* ref
- R3 [satisfied] check:package-manifests — the generator runs from pose-mcp/ and produces the four expected files against the published v0.20.3 checksums

### Known gaps
- The clean-host brew/winget installs have still never executed. This cut is
  their first opportunity, and the matrix runs on macOS and Windows only.

---

## 7. Final Report

### Delivered scope
The Homebrew formula and WinGet manifests are published as release assets, and
the clean-host verification has a trigger that fires and a command that runs.

### Files and modules changed
- .github/workflows/release.yml
- .github/workflows/package-channels.yml

### Validation executed
- Command: the manifest generator against published v0.20.3 checksums
- Result: four files produced, named as documented

### Residual risks
- Nothing verifies that the documentation's commands match the assets actually
  published; the two drifted apart unnoticed for the project's whole history.

### Follow-ups

- [covered: pose-docs-asset-parity] Assert in CI that every install command in docs-site/docs/package-channels.md resolves to an asset the release actually publishes. The drift that made pose.rb a 404 went unnoticed because nothing compared the two. (owner:@pose-maintainers crit:medium review:2026-10-02)
