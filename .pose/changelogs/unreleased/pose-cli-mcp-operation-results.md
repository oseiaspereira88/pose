---
spec: pose-cli-mcp-operation-results
category: changed
breaking: false
---

`pose close spec:<slug> --apply|--resume --json` now writes exactly one JSON
document, the operation result: `operation`, `outcome` (`completed`,
`already-satisfied`, `waiting`, `blocked`, `conflict`, `failed`), the step it
stopped at, the steps it ran, the pending criteria, the next action, the
effects the domain confirmed, the steps left unconfirmed after a failure and
a typed cause. It used to print text with `--json`. The human output reads the
same result, and a call that runs no step on a spec already closed reports
`closeout_plan.result=already-satisfied` instead of `closed`, because it did
not verify anything; `pose closeout-check` does.

`pose continuous-closeout status --json` writes an operation result instead
of ignoring the flag; the exit keeps its meaning.

Stricter parsers, classified for a minor release: `pose specs` refuses an
unknown flag, a missing value, an unparseable or negative `--since` and a
non-positive `--recent` with exit 2 instead of listing every spec, and MCP
`pose_list_specs` refuses the same values; `pose close` refuses two different
modes and `--plan` with `--digest`, `--reviewer` or `--skip-evidence`;
`pose start` refuses two of `--apply`, `--status` and `--cancel`, `--digest`
outside `--apply` and `--cancel --json`. A repeated `pose start --apply
--json` returns the start record as JSON instead of text. `pose specs facts`
refuses unknown arguments and reads `--json` in any position; `pose_list_specs`
also refuses unknown arguments, an explicit `0` for `recent` or `limit`, an
explicit `null` and an empty `since`, and keeps the advertised `sort: "slug"`.
`pose close <scope> --json` (no plan mode) is refused instead of ignored. A
`pose start --apply` that fails after recording its baseline reports it as a
confirmed effect and says the same apply resumes it; repeating it now finishes
from its record a transition that was already published, while a spec that
has not moved is judged again by the current plan and its restrictions. With `--json`, a failure of `start`, `close --plan`, `specs` or
`specs facts` is one JSON operation result instead of an empty stdout.

An MCP gate (`pose_check`, `pose_lint_spec`, `pose_skills_check`) stopped by
its deadline or a cancellation now returns an error result with
`error_code: execution_inconclusive`, the cause and the partial output, where
it used to return `passed: false`; a stopped gate no longer holds the call
while a process it started keeps the output open.

The usage journal counts a close that stops on a reviewer as `waiting`, not
`failed`, and marks derived signal events `duration_unknown` instead of 0 ms;
`pose usage` reports `waiting` and `duration_samples` and shows an unmeasured
P95 as `-`.
