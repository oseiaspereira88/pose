---
type: decision-log
slug: escalation-validate-native
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-14
last_reviewed_at: 2026-08-14
expires_at: 2026-11-12
source_refs:
  spec: "pose-component-aware-review-plans"
  workflow: "recurrence-escalation"
  commands: ["pose recurrence-check --tolerant --window-days 30 --threshold 3", "pose stats workflows --since-days 30", "pose stats tasks --since-days 30 --json", "pose validate --strict --module pose-mcp --json .pose/results/delivery-validation.json --report"]
  external_sources: []  # [{url: "", accessed_at: "YYYY-MM-DD"}]
---

# decision-log: escalation-validate-native

## Context

The 30-day recurrence projection crossed its threshold for `validate-native`
with 13 retained failures. Four recent failures came from
`.qwen/worktrees/pr15-review`; module discovery recursively treated that
agent-local worktree as another product module. The same projection flagged
three `test-plan-baseline-pose-mcp-active-context` failures from 2026-08-06,
but those were implementation retries interleaved with passes and ended with
strict PASS on the same day, so they are not uncovered incidents.

## Current state

Commit `3177e80` excludes `.qwen` from validation discovery and adds a
regression. Commit `39bf6ac` applies the same boundary to repository indexing.
Strict module validation now passes and its canonical JSON is bound to both
release specs. A 45-day intervention is registered in
`.pose/reports/history/interventions.jsonl`; the historical failures remain
append-only, so `recurrence-check` will continue to report them until the
window ages out.

## Next checks

- After 2026-09-28, run `pose recurrence-effect --min-sample 3` and record a
  `keep`, `adjust`, or `discard` decision.
- Expect no new `validate-native` failure whose `report_path` is below
  `.qwen/worktrees/`; any such failure reopens the discovery boundary.
- Keep `pose recurrence-check --tolerant --window-days 30 --threshold 3` in
  review closeout. Success means zero new uncovered failures, not deletion of
  immutable historical failures.

## Risks

The pre-intervention sample mixes ordinary development failures with the
recursive-worktree defect, so effectiveness may be inconclusive. Classify
future records by execution root and cause, preserve all outcomes, and do not
claim success before the observation window completes.

## Next owner

`@pose-maintainers` on or after 2026-09-28.

## References

- `spec:pose-component-aware-review-plans`
- `knowledge:pr15-component-aware-review-provenance`
- `.pose/workflows/recurrence-escalation.md`
- `.pose/reports/history/standard-validate-native.jsonl`
- `.pose/reports/history/standard-test-plan-baseline-pose-mcp-active-context.jsonl`
- Commits `3177e80` and `39bf6ac`
