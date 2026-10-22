---
spec: pose-distributed-instructions-parity
category: changed
breaking: false
---

The instructions POSE distributes now describe the engine as it is. The feature workflow and
the delivery-evidence rule, in English and Portuguese and in every embedded mirror, ask for
`pose assess discover --if-stale`, the way the skills and `AGENTS.md` already did, so a session
or a closeout no longer rescans a component whose inputs did not change.

`pose suggest` resolves domains through one taxonomy (`go`, `golang` and `backend-go` reach the
same rules; so do the React and Kubernetes spellings), tells you when it used an alias, and
reports a domain the task map has no rules for instead of silently borrowing another domain's.
A `backend` module found in the repository map is routed to the Go rules only when it is a Go
module. Each rule it names carries its availability in `rules_status` and in the human output:
installed (with its path), `extension-needed` (with the `pose extension install` command) or
unavailable; it no longer announces a rule file that is not on disk, and it installs nothing.

`pose migrate v7 --apply` is refused with the current fact (no apply is available in this
engine; `--dry-run` writes nothing) and no longer promises a future 7.0 apply.
