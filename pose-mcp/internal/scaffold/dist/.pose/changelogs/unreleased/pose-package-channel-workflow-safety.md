---
spec: pose-package-channel-workflow-safety
category: security
breaking: false
refs:
---

The package-channel gate no longer checks out an event-supplied ref. It gained
a `workflow_run` trigger in the same cycle that hardened the verification
workflow against exactly that pattern, so the Scorecard Dangerous-Workflow
finding relocated instead of closing. Both workflows now resolve the release
tag once, refuse anything that is not one, and pass it to later steps as an
environment variable rather than as interpolated template text.
