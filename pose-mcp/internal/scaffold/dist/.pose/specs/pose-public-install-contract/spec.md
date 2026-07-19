---
slug: pose-public-install-contract
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-version-contract
priority: 3
---

# Spec: Real public install contract

## 1. Intent

### Goal
publish a stable, copyable and verified download/install path with no owner or repository placeholders.
### Business value
Removes the highest-friction break in the freemium conversion path.
### Constraints
- Support current Linux, macOS and Windows artifacts and verify integrity before execution.
### Non-goals
- Deliver every package-manager channel.

## 2. Requirements

### Functional
- R1: Documentation shall resolve a released asset for every supported OS and architecture.
- R2: The install flow shall verify integrity before placing the binary on `PATH`.
- R3: A clean-host test shall install, initialize and run `pose doctor --json` plus `pose check --strict`.

### Non-functional
- Keep bootstrap small, auditable and account-free.

### Security
- Never recommend piping unverified network content to a privileged shell.

### Compatibility
- Document supported shells, operating systems and minimum dependencies.

## 3. Technical Plan

### Affected areas
- README, quickstart, release assets, installer E2E and `pose-action` examples.

### API/contract changes
- The README quickstart becomes a tested public entry contract.

### Data/storage changes
- No repository data changes beyond ordinary `pose install`.

### Technical risks
- Release asset naming drift can silently break copyable commands.

### Primary references
- [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)
- [Sigstore documentation](https://docs.sigstore.dev/)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions): quickstart assumed a binary already on `PATH`; `<owner>/<repo>` placeholders in `docs-site/docs/ci.md` and `pose-action/README.md`; stale `rev: v0.2.0` pre-commit pin; Windows shipped `tar.gz`.

### Implementation
- [x] Specify asset resolution and verification per platform: README quickstart documents `pose_<version>_<os>_<arch>` (`tar.gz` Linux/macOS, `zip` Windows via GoReleaser `format_overrides`) with mandatory `checksums.txt` verification before the binary reaches `PATH`; explicit "never pipe downloaded scripts into a shell" guidance. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- [x] Replace all action and repository placeholders with released coordinates: `oseiaspereira88/pose` in `docs-site/docs/ci.md` and `pose-action/README.md`; pre-commit pinned to `v0.9.0`; production guidance pins tags/SHAs. ([reference](https://docs.sigstore.dev/))
- [x] Add clean-container and clean-VM smoke tests for the published quickstart: `tests/install/run.sh` now packs the built binary as a release-named archive, verifies its checksum, extracts onto a restricted `PATH` and runs `pose install` + `pose doctor --json` + `pose check --strict` on a fresh repository; `TestPublicInstallContract` pins README/docs to `version.ReleaseBase()` and the GoReleaser template. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Install|Doctor'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.sigstore.dev/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-verified-public-install-contract.md` (Accepted): documented download-verify-install contract tested in CI, over `curl | bash` (rejected on security grounds) and package-manager-first (owned by `pose-package-manager-distribution`); asset naming is a public contract; verification precedes execution; docs are pinned by failing tests.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Install|Doctor'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-public-install-contract --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/version/ ./internal/cli/ -run 'Version|Install|Doctor|Public' -count=1` — SUCCESS.
- `bash tests/install/run.sh` — PASS (includes the new verified-download scenario with checksum verification, restricted `PATH`, `pose doctor --json` and `pose check --strict`).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-public-install-contract --ready-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Copyable, checksum-verified quickstart for Linux, macOS (bash/zsh) and Windows
(PowerShell) pinned to the release base; Windows `zip` archives; placeholder
removal in CI docs and GitHub Action; pre-commit pinned to an immutable tag;
`TestPublicInstallContract` guarding version, asset naming and placeholder
regressions; clean-host E2E exercising verify → install → doctor → gate; ADR
recorded.

### Residual risks

- The quickstart pins an explicit version instead of resolving "latest";
  stale pins are caught by the contract test at the next version bump, but a
  user can still copy commands from an outdated page cached elsewhere.

### Follow-ups

- [covered: pose-package-manager-distribution] Homebrew/Scoop/Winget/Nix channels.
- [covered: pose-release-signing] Sigstore signatures for release artifacts beyond SHA-256 checksums.
- [covered: pose-upgrade-compatibility-lab] In-place upgrade verification across released versions.
