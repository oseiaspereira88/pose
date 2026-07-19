---
spec: pose-public-install-contract
category: added
breaking: false
refs:
---

The quickstart now documents a real, copyable download-verify-install path for
Linux, macOS and Windows with mandatory checksum verification before the
binary reaches `PATH`. Windows releases ship as `zip`, owner/repo placeholders
are gone from the CI docs and GitHub Action, and a contract test plus a
clean-host E2E keep the published commands aligned with the released assets.
