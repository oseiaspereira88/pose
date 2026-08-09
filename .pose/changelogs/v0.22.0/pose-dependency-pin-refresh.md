---
spec: pose-dependency-pin-refresh
category: added
breaking: false
refs:
---

Pinned dependencies now have a refresh path. Making every action, container base
and Python package immutable closed 84 Scorecard findings and, deliberately,
traded silent drift for silent staleness: an upstream security fix arrived only
when someone updated a SHA by hand. Dependabot now covers all four pinned
ecosystems with grouped monthly updates, and `scripts/refresh-action-runtimes.sh`
turns the runtime-manifest edit that a bump forces into one idempotent command.
