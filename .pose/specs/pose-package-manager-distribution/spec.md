---
slug: pose-package-manager-distribution
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-public-install-contract, pose-release-signing, pose-release-compatibility-matrix
priority: 25
---

# Spec: Supported package-manager distribution

## 1. Intent

### Goal
publish authenticated releases through maintained macOS, Windows and Linux-friendly channels.
### Business value
Removes manual binary placement from mainstream freemium onboarding.
### Constraints
- Every channel consumes the same signed artifacts and documents rollback.
### Non-goals
- Maintain unofficial channels without a service level.

## 2. Requirements

### Functional
- R1: Homebrew and at least one Windows channel shall install the authenticated release.
- R2: Metadata shall update only after release verification passes.
- R3: A clean-host matrix shall install, run doctor/check and uninstall each package.

### Non-functional
- Measure publication lag and expose support status.

### Security
- Pin digests and use least-privilege publisher credentials.

### Compatibility
- Channel versions shall follow release and schema policy.

## 3. Technical Plan

### Affected areas
- Release automation, package manifests, docs and clean-host tests.

### API/contract changes
- Publish support tiers, update latency and deprecation per channel.

### Data/storage changes
- Retain generated manifests and publication results.

### Technical risks
- Compromised channel credentials can redirect trusted package names.

### Primary references
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [WinGet package manifests](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook).

### Implementation
- [ ] Select channels and ownership/service-level policy. ([reference](https://docs.brew.sh/Formula-Cookbook))
- [ ] Generate manifests from verified release metadata. ([reference](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest))
- [ ] Run clean-host install, doctor, check, upgrade and uninstall tests. ([reference](https://docs.brew.sh/Formula-Cookbook))

### Validation
- [ ] Run `pose check --strict` and retain evidence. ([reference](https://docs.brew.sh/Formula-Cookbook))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-package-manager-channels-generated-not-hosted.md` (Accepted): manifests are generated deterministically from `checksums.txt` + release tag (`pose release-package-manifests`), not hosted in a maintained tap/registry — Homebrew installs directly from the formula attached to the GitHub release (`brew install --formula <url>`, zero publication lag, no upstream review dependency); WinGet ships as a generated manifest artifact submitted to `winget-pkgs` by a maintainer (non-zero, tracked publication lag). Rejected: owning a Homebrew tap or submitting to homebrew-core (extra sync/credential surface, upstream review dependency not yet justified by volume); adding Scoop/Nix now (out of scope for R1's "at least one Windows channel", left as a small future extension of the same generator).

## 6. Validation

**Strategy:** validate the deterministic generator with unit and negative/security cases (missing-checksum, malformed-checksums-line, malformed-version), a release-pipeline placement check (R2), and a clean-host install/doctor/uninstall matrix (R3).

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'Checksum|Homebrew|WinGet|CmdReleasePackageManifests' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-package-manager-distribution --ready-check`.

### Requirement trace
- R1 [satisfied] Homebrew (formula-URL install) and WinGet (generated manifest set) both authenticate the same signed release artifacts; check:test (TestHomebrewFormulaDeterministicContent, TestWinGetManifestsDeterministicContent, TestHomebrewFormulaMissingChecksumFails, TestWinGetManifestsMissingChecksumFails)
- R2 [satisfied] manifest generation is sequenced in `.github/workflows/release.yml` strictly after the compatibility gate, security gate, GoReleaser build/sign/SBOM and `tests/release/verify.sh`; check:doc (`.github/workflows/release.yml` "Package-manager manifests" step, ADR `2026-07-19-package-manager-channels-generated-not-hosted.md`)
- R3 [satisfied] `.github/workflows/package-channels.yml` runs a macOS/Windows clean-host matrix (install, `pose doctor --json`, uninstall) on every published release; check:doc (`.github/workflows/package-channels.yml`) — not executable in this sandbox (no brew/winget/network), deferred to CI on the first tagged release (open follow-up)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'Checksum|Homebrew|WinGet|CmdReleasePackageManifests' -v -count=1` — SUCCESS (16 tests: valid/blank-line/malformed checksums parsing; deterministic formula and manifest content across repeated calls; missing-checksum rejection for both channels; CLI happy path, malformed-version rejection, missing-checksums-file rejection, usage error).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold` (embedded scaffold mirrors the new ADR/doc/changelog files).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-package-manager-distribution --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- R2 (metadata updates only after verification passes): enforced structurally, not just by convention — the `Package-manager manifests` step in `.github/workflows/release.yml` is sequenced after the compatibility gate, the security gate, GoReleaser build/sign/SBOM and `tests/release/verify.sh`; any prior step failing halts the job before generation runs.
- R3 (clean-host matrix): `.github/workflows/package-channels.yml` runs a macOS/Windows matrix on every published release — install via the real channel, `pose doctor --json`, uninstall. Not executable in this sandbox (no brew/winget/network); deferred to CI execution on the first tagged release, consistent with how `pose-slsa-provenance` and `pose-reproducible-release-verification` handled sandbox-unavailable infrastructure. Follow-up opened to confirm the first real run.

## 7. Final Report

- Delivered scope: deterministic Homebrew formula + WinGet manifest generator (`pose release-package-manifests`) driven by `checksums.txt` and the release tag; wired into `release.yml` strictly after every existing verification step (R2); a clean-host `package-channels.yml` CI matrix for macOS (Homebrew) and Windows (WinGet) covering install/doctor/uninstall (R3); support-tier and rollback documentation (`docs-site/docs/package-channels.md`) addressing the non-functional publication-lag/support-status requirement; ADR recording the generated-not-hosted channel decision.
- Residual risk: compromised channel credentials could redirect trusted package names — mitigated by the generator having no persistent credentials of its own (it only reads `checksums.txt`, already produced by the signed/verified release) and by WinGet publication requiring a maintainer-reviewed PR into `winget-pkgs` rather than an automated push; Homebrew has no credential at all since it installs directly from a release-pinned URL. `winget-pkgs` submitter account compromise remains an upstream risk outside this repository's control.
- Follow-ups: see below.

### Follow-ups

- [open] Confirm the first real `package-channels.yml` clean-host run on the first tagged release (brew/winget unavailable in this sandbox). (owner:@pose-maintainers crit:medium review:2026-08-19)
- [open] Submit the first generated WinGet manifest to `winget-pkgs` and record the observed publication lag in `package-channels.md`. (owner:@pose-maintainers crit:low review:2026-08-19)
- [open] Revisit Homebrew tap ownership once install volume justifies the maintenance cost over the formula-URL install. (owner:@pose-maintainers crit:low review:2026-10-19)
