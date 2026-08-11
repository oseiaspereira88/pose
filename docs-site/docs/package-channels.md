# Package-manager channels

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE 1.0.x

Every channel below installs the exact same signed release artifact used by
the [verified install contract](quickstart.md#install) — no channel ever
builds from source or repackages independently. Manifests are generated
deterministically from `checksums.txt` and the release tag by
`pose release-package-manifests`, run in CI only after the compatibility
gate, the security gate and artifact-identity verification have all passed
(spec `pose-package-manager-distribution`, R2).

## Channels and support tiers

The supported install path on every platform is the verified download: fetch the
archive and `checksums.txt` from the release, verify the checksum, extract — see
[Quickstart](quickstart.md#install). The package-manager channels below are
additive, and neither is currently a one-command install.

| Channel | Format | Publication mechanism | Publication lag | Support tier |
|---|---|---|---|---|
| Homebrew | `pose.rb` formula | Attached to the GitHub release as an asset. **No install channel:** Homebrew requires a formula to be in a tap, and installing one from a path or URL is rejected — so the published formula is consumable only by a tap that does not exist yet | n/a | Generated and install-tested on every tagged release, through a throwaway tap on the clean-host matrix; not consumable by an end user |
| WinGet | 3-file manifest set (`version`/`installer`/`locale.en-US`) | Generated in CI and attached to the release; a maintainer submits it as a PR to `microsoft/winget-pkgs` | Days, gated by upstream Microsoft review — tracked per release in the closing spec's follow-ups until publication is automated | Maintained: manifest generation and local install exercised on every tagged release; upstream publication is a manual, tracked step |

Install commands:

```bash
# Every platform: the verified download (see the install contract)

# WinGet (Windows), once published to winget-pkgs
winget install Harne8.Pose
```

!!! warning "Homebrew is not an install channel"

    Earlier versions of this page offered
    `brew install --formula <url>`. Homebrew rejects that: a formula must be in
    a tap, and installing from a path or a URL is unsupported. The formula is
    still generated, published and install-tested every release, so a tap can
    be stood up without regenerating anything — but until one exists there is
    no `brew` command that installs POSE, and the verified download is the
    supported path on macOS.

## Verification

The `Package channels` CI workflow (`.github/workflows/package-channels.yml`)
installs, runs `pose doctor --json` and uninstalls through each channel on
an unmodified macOS and Windows runner for every published release (spec
`pose-package-manager-distribution`, R3). On macOS the formula is installed
through a throwaway local tap, since that is the only supported way to install
a formula file — the artifact under test is the published `pose.rb`, not the
tap. A channel that fails this matrix blocks that release's support-tier claim,
not the release itself — package channels are additive to the verified
download-and-checksum contract, never a replacement for it.

That matrix ran for the first time on v0.21.0 and exposed two install-path
defects: Homebrew requires a tap, and WinGet requires local-manifest policy to
be enabled. The repaired clean-host run (`31240578941`) then passed artifact
validation, install and `pose doctor --json` on both macOS and Windows. This is
why the formula remains release-tested while this page still does not present
it as an end-user Homebrew channel: a throwaway CI tap is proof of the artifact,
not a maintained public tap.

## Rollback

Every channel installs a specific pinned version. To roll back:

- **Homebrew:** not applicable — there is no install channel to roll back.
  Re-run the verified download against the prior tag.
- **WinGet:** `winget install Harne8.Pose --version <prior-version>`, or
  uninstall and reinstall from the prior release's manifest artifact.

`pose upgrade` handles the repository contract/schema side of moving
between versions once the new binary is on `PATH`; channel rollback only
changes which binary is installed.
