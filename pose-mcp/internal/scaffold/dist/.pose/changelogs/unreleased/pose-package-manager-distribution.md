---
spec: pose-package-manager-distribution
category: added
breaking: false
refs:
---

Releases now generate a deterministic Homebrew formula and WinGet manifest
set (`pose release-package-manifests`) from the same signed `checksums.txt`
every other channel already trusts. Homebrew installs directly from a
release-attached formula URL with zero publication lag; WinGet ships as a
generated manifest artifact for submission to `winget-pkgs`. A clean-host CI
matrix installs, runs `pose doctor` and uninstalls through each channel on
every published release, and support tiers/rollback are documented at
`docs-site/docs/package-channels.md`.
