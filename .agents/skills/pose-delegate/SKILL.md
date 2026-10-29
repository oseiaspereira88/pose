---
name: pose-delegate
description: Delegate a POSE review, adjudication, smoke run or implementation task through the native execution surface; follow its routing, conclusions and next actions.
when_to_use: Handing bounded work to another agent or continuing a recorded delegated execution. Use a review purpose for governance evidence and task for general work.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, execute
---

# Skill: pose-delegate

## Required reading

Read [AGENTS.md](../../../AGENTS.md) and the applicable module instructions.
Resolve the task with `pose context --task <ref> --json`; use its canonical
authority. Consult the delegation section of [POSE.md](../../../POSE.md) for
policy, containment and signing configuration.

## Steps

1. Choose `review`, `adjudicate` or `smoke` when the conclusion must satisfy a
   governed scope. Validate and commit the subject, seal its bundle from the
   implementing session, and inspect `pose review brief <bundle-id> --kind
   <purpose>`. Let POSE generate the reviewer's prompt from the sealed subject;
   do not replace it with a prompt written by the implementer. Put additional
   context in a labelled `--note-file`.
2. Choose `task` for implementation, investigation or other general work.
   Supply that task's prompt file, explicit directory and the access it needs.
   A task's success is never a review, attestation or evidence of delivery.
3. Inspect `pose delegate agents --json` and preview the chosen operation with
   `--dry-run --json`. Prefer automatic routing against the implementation
   identity sealed in the bundle. A manually chosen agent does not establish
   independence by its name; use the recorded vendor/model and proof.
4. Run `pose delegate <purpose> <bundle-id>` for a governed review. Use native
   `ls`, `watch`, `wait`, `resume` and `handoff` to follow its recorded execution.
   Respect recorded limits, resets, budgets and containment failures. Do not
   launch repeated fresh runs to hide a failed or rejecting attempt.
5. Read the run and its conclusion. A completed process alone is no approval.
   With delegated review adopted and its issuer configured, the engine records
   a complete verified conclusion automatically. Otherwise inspect the refusal
   and missing prerequisites. Never write the reviewer's attestation yourself
   or represent a local executor signature as a vendor receipt or a person.
6. Answer a rejection with a change and current validation, then seal a new
   bundle for the next round. Do not seek an approval by rerunning an unchanged
   rejected subject with another reviewer. Keep the rejecting record.
7. Follow structured `next_actions` using the command and preconditions the
   engine returns. Preview actions before applying where supported. A business
   decision, mandatory-human criterion, unresolved disagreement, exhausted
   budget or unavailable independent agent goes to the native action request's
   authorized recipient. Do not answer for a person or weaken the policy.
8. Verify the scope with `pose review verify`, `pose review-check` and
   `pose closeout-check`. Close through the native command only when its gates
   pass. When delegation is explicitly off, preserve the disclosed
   `delegated-review-disabled` limitation; mandatory-human and a stronger
   protected baseline still apply.

## Output requirements

Report the scope/bundle, selected agent, achieved independence, run/attempt id,
verified conclusion and next action. Distinguish execution success, recorded
rejection, refused conclusion and approved review. Include an escalation's
request id when work needs the authorized person.
