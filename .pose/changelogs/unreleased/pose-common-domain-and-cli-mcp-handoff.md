---
spec: pose-common-domain-and-cli-mcp-handoff
category: changed
breaking: false
---

MCP tools no longer start the POSE binary for the two read-only operations whose
semantics the CLI package owns: `pose_suggest` and `pose_get_followups` run the
same command function in process, with the same results and errors, the same
deadline and cancellation handling (an execution failure, never a verdict) and no
usage side effect. The gates (`pose_check`, `pose_skills_check`, `pose_lint_spec`)
keep the process boundary. Every MCP tool now has a declared route class, kept
equal to the catalog by a test. `pose_closeout_state` with `include_plan` also
returns a typed handoff when the plan has steps left: the `pose close --apply`
invocation as structured arguments, the full plan digest and the other digests it was computed from, the target project and root, the
authority it needs and when it stops being valid; it executes nothing and is not
an approval.
