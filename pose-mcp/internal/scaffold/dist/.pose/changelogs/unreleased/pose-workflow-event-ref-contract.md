---
spec: pose-workflow-event-ref-contract
category: added
breaking: false
refs:
---

A contract test now fails the build when a workflow triggered by `workflow_run`
or `pull_request_target` checks out an event-supplied ref or interpolates one
into a shell script. The pattern was removed from one workflow and reintroduced
in another the same day, by a sibling change, with the corrected form already in
the repository — review read each diff on its own merits and both were
individually defensible. Validated `if:` guards and `env:` bindings stay
allowed.
