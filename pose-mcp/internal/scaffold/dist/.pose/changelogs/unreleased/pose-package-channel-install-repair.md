---
spec: pose-package-channel-install-repair
category: fixed
breaking: false
refs:
---

The documentation no longer offers `brew install --formula <url>`, which
Homebrew rejects: a formula must be in a tap, and installing one from a path or
a URL is unsupported. The formula is still generated, published and
install-tested every release — through a throwaway tap on the clean-host
matrix — so a tap can be stood up later without regenerating anything, but
until one exists there is no `brew` command that installs POSE and the verified
download is the supported path on macOS. The WinGet leg now enables
local-manifest installation, which is off by default, and validates the manifest
set before installing.
