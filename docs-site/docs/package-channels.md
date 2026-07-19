# Package-manager channels

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

Every channel below installs the exact same signed release artifact used by
the [verified install contract](quickstart.md#install) — no channel ever
builds from source or repackages independently. Manifests are generated
deterministically from `checksums.txt` and the release tag by
`pose release-package-manifests`, run in CI only after the compatibility
gate, the security gate and artifact-identity verification have all passed
(spec `pose-package-manager-distribution`, R2).

## Channels and support tiers

| Channel | Format | Publication mechanism | Publication lag | Support tier |
|---|---|---|---|---|
| Homebrew | `pose.rb` formula | Attached to the GitHub release; `brew install --formula <url>` reads it directly — no upstream tap review | None — available the moment the release publishes | Maintained: exercised on every tagged release by the clean-host matrix |
| WinGet | 3-file manifest set (`version`/`installer`/`locale.en-US`) | Generated in CI as a release artifact (`pose-package-manifests`); a maintainer submits it as a PR to `microsoft/winget-pkgs` | Days, gated by upstream Microsoft review — tracked per release in the closing spec's follow-ups until publication is automated | Maintained: manifest generation exercised on every tagged release; upstream publication is a manual, tracked step |

Install commands:

```bash
# Homebrew (macOS, Linux)
brew install --formula https://github.com/oseiaspereira88/pose/releases/download/vX.Y.Z/pose.rb

# WinGet (Windows), once published to winget-pkgs
winget install Harne8.Pose
```

## Verification

The `Package channels` CI workflow (`.github/workflows/package-channels.yml`)
installs, runs `pose doctor --json` and uninstalls through each channel on
an unmodified macOS and Windows runner for every published release (spec
`pose-package-manager-distribution`, R3). A channel that fails this matrix
blocks that release's support-tier claim, not the release itself — package
channels are additive to the [verified download-and-checksum
contract](quickstart.md#install), never a replacement for it.

## Rollback

Every channel installs a specific pinned version. To roll back:

- **Homebrew:** re-run `brew install --formula` against the prior tag's
  `pose.rb` URL (release archives and formulas for all supported prior
  versions remain attached to their GitHub releases).
- **WinGet:** `winget install Harne8.Pose --version <prior-version>`, or
  uninstall and reinstall from the prior release's manifest artifact.

`pose upgrade` handles the repository contract/schema side of moving
between versions once the new binary is on `PATH`; channel rollback only
changes which binary is installed.
