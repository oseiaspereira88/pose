# ADR: Package-manager channels — generated manifests, not a hosted tap/registry

## Status
Accepted (2026-07-19) — spec `pose-package-manager-distribution`

## Context

The [verified public install contract](../adr/2026-07-19-verified-public-install-contract.md)
covers direct download + checksum verification but explicitly deferred
package-manager channels as a separate spec. Mainstream freemium onboarding
still expects `brew install` and a native Windows package manager. Any
channel must consume the exact same signed release artifacts as the direct
download path (R1) and must never update its metadata ahead of release
verification (R2) or without a clean-host proof that install/doctor/check/
uninstall actually work (R3).

Alternatives considered:

1. **Own and maintain a Homebrew tap repository / submit to homebrew-core.**
   A dedicated tap gives a friendlier `brew install pose` (no `--formula`
   URL), but adds a second repository to keep in sync with every release,
   plus its own CI and credentials. `homebrew-core` submission adds an
   upstream review dependency for a project that doesn't yet meet its
   notability/maintenance bar.
2. **Scoop/Nix as additional channels.** Real candidates for a later spec,
   but doubling the channel count doubles the clean-host matrix and the
   manifest formats to keep deterministic; out of scope for this spec's
   "at least one Windows channel" requirement (R1).
3. **Generate manifests deterministically from the same release metadata
   every other channel already trusts (`checksums.txt` + the release tag),
   publish the Homebrew formula directly as a release asset (installable
   with `brew install --formula <url>`, no tap needed), and generate the
   WinGet manifest set as a release artifact for submission to
   `winget-pkgs`.**

## Decision

Option 3.

- **Generator, not a hosted service:** `pose release-package-manifests
  --version --checksums --out` (`internal/cli/release_manifests.go`) is a
  pure function of `checksums.txt` and the release version — same input,
  byte-identical output, verified by
  `TestHomebrewFormulaDeterministicContent` /
  `TestWinGetManifestsDeterministicContent`. No network access, no
  credentials, no tap repository to keep in sync.
- **Placement in the release pipeline enforces R2:** the generation step in
  `.github/workflows/release.yml` runs after the compatibility gate, the
  security gate, GoReleaser's build/sign/SBOM, and
  `tests/release/verify.sh` — any prior failure halts the job before
  manifests are produced, so channel metadata can never advance ahead of a
  failed release.
- **Homebrew ships as a formula-URL install (`brew install --formula
  <release-url>/pose.rb`), not a hosted tap.** Zero publication lag (the
  formula is a release asset, available the instant the release publishes)
  and zero upstream review dependency, at the cost of a longer install
  command than a tap would give. Revisit tap ownership once install volume
  justifies the maintenance cost.
- **WinGet ships as a generated manifest artifact submitted to
  `winget-pkgs` by a maintainer**, since WinGet has no self-hosted
  equivalent to a formula URL — the winget-pkgs community repository is the
  only distribution path Microsoft's client trusts. Publication lag is
  therefore non-zero and tracked per release as an open follow-up until
  submission is automated.
- **R3 clean-host proof:** `.github/workflows/package-channels.yml` runs a
  macOS/Windows matrix on every published release — installs through the
  real channel, runs `pose doctor --json`, uninstalls. A failure here is a
  support-tier signal (documented in `package-channels.md`), not a release
  blocker; it never gates the release job itself, since by the time it runs
  the release already published successfully.
- **Support tiers and rollback are documented, not just implied:**
  `docs-site/docs/package-channels.md` states publication mechanism, lag
  and rollback per channel per the non-functional requirement ("measure
  publication lag and expose support status").

## Consequences

- Positive: manifest generation has zero new attack surface beyond parsing
  `checksums.txt` — no credentials, no persistent hosted state, fully unit
  tested without brew/winget/network in the loop.
- Positive: Homebrew users get an immediately-working install command with
  no publication lag or upstream dependency.
- Negative: `brew install --formula <url>` is less discoverable than `brew
  install pose`; a hosted tap remains a future option once volume
  justifies it (tracked as a follow-up on this spec).
- Negative: WinGet publication lag is real and manual until a bot-driven
  `winget-pkgs` submission exists; tracked as an open follow-up.
- Neutral: adding a third channel (Scoop, Nix) is a small extension of the
  same generator — one more render function and clean-host job, not a new
  architecture.
