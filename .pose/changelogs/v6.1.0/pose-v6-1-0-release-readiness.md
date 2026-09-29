---
spec: pose-v6-1-0-release-readiness
category: changed
breaking: false
---

Prepare POSE 6.1.0 with aligned public version metadata and an authenticated upgrade path from 6.0.4, pinned by the SHA-256 of its published `checksums.txt`. The release introduces no contract or schema; it is a minor because the machine channel adds flags to seven gates, moves three gates' findings from stderr to stdout, and deprecates `pose validate --json <path>`.
