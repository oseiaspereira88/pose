---
spec: pose-release-workflow-hardening
category: security
breaking: false
refs:
---

Release workflows tightened ahead of the repository going public. The
independent verification no longer checks out an event-supplied ref: it accepts
only a release tag and refuses anything else before executing repository
content. The release workflow grants `contents: read` at the top level, with the
write scopes it needs declared on the publishing job.
