---
spec: pose-governed-effect-enforcement
category: added
breaking: false
refs:
---

Action requests can now restrict transitions, opt-in with `agency_readiness_version: 1` in the review policy. Once adopted, `pose close` refuses with the request's cause while it restricts closeout, `closeout-check` and `pose_closeout_state` report the typed `action-request-pending` diagnostic and an `answer-action-request` next step, continuous closeout cannot complete, `pose start --apply` refuses a restricted start (and a preview taken before the request is stale), and `pose release prepare` refuses while a request restricts the release. Each gate recomputes when it runs; without adoption nothing changes.
