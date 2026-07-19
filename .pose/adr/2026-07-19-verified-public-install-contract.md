# ADR: Verified public install contract

## Status
Accepted (2026-07-19) — spec `pose-public-install-contract`

## Context

The public install path had placeholders (`<owner>/<repo>` in the CI docs and
`pose-action` README, a stale `rev: v0.2.0` pre-commit pin) and the README
quickstart assumed the binary was already on `PATH` without saying where to
get it or how to verify it. Release archives were `tar.gz` for every OS,
including Windows, where native tar handling is still inconsistent across
supported shells. Nothing tested that documented commands match the real
release asset names, so a GoReleaser template change could silently break
every copyable command.

Alternatives considered:

1. **`curl | bash` one-liner installer** — lowest friction, but executes
   unverified network content in the user's shell; rejected by the security
   requirement.
2. **Package managers first (Homebrew/Scoop/Winget/Nix)** — the right
   long-term channels, but each adds an external publication dependency; they
   are owned by `pose-package-manager-distribution` (Adoption and DX roadmap).
3. **Documented download-verify-install contract, tested in CI** — copyable
   commands per platform with mandatory checksum verification, guarded by a
   contract test and a clean-host E2E scenario.

## Decision

Option 3. The install contract:

- **Asset naming is a public contract:** `pose_<version>_<os>_<arch>`,
  `tar.gz` for Linux/macOS and `zip` for Windows (GoReleaser
  `format_overrides`), across `linux`/`darwin`/`windows` ×
  `amd64`/`arm64`, plus `checksums.txt` (SHA-256) and `install.sh` in the
  release.
- **Verification precedes execution:** the README quickstart downloads the
  archive and `checksums.txt`, verifies (`sha256sum --check` /
  `shasum -a 256 -c` / `Get-FileHash`) and only then places `pose` on `PATH`.
  The docs never recommend piping downloaded content into a shell;
  `install.sh` is a local bootstrap run beside the verified binary.
- **Docs are tested:** `TestPublicInstallContract`
  (`internal/version/contract_test.go`) fails when the README pinned version
  diverges from `version.ReleaseBase()`, when asset names diverge from the
  GoReleaser template, when the Windows zip override disappears or when
  owner/repo placeholders reappear in `docs-site/docs/ci.md` or
  `pose-action/README.md`.
- **Clean-host proof:** the installer E2E (`tests/install/run.sh`) packs the
  built binary as a release-named archive, verifies its checksum, extracts it
  onto a restricted `PATH` and runs `pose install`, `pose doctor --json` and
  `pose check --strict` on a fresh Git repository (R3).
- **Supported environments are documented:** bash/zsh + coreutils or
  PowerShell; Git required; no Bash/Python/Node at runtime.

## Consequences

- Positive: the freemium entry path has no placeholder dead-ends; every
  copyable command is exercised or pinned by a failing test.
- Positive: Windows users get a native `Expand-Archive` flow instead of tar.
- Trade-off: bumping the release version now requires updating the README pin
  (one commit with `internal/version` and `server.json`, enforced by tests) —
  intentional, since stale install commands are worse than the edit cost.
- Residual: signatures and provenance (Sigstore, SLSA, SBOM) are deliberately
  out of scope here; they are owned by the supply-chain-trust roadmap
  (`pose-release-signing`, `pose-slsa-provenance`, `pose-cyclonedx-sbom`).
