---
slug: pose-package-channel-workflow-safety
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-release-workflow-hardening
priority: 1
components: release
delivers: governance:package-channel-workflow-safety
---

# Spec: The channel gate stops re-opening the hole its sibling closed

## 1. Intent

### Goal
Remove the event-supplied ref from `package-channels.yml`'s checkout and from
every script it feeds, applying the resolution `verify-release.yml` already
uses, and close the same interpolation left behind there.

### Business value
`pose-release-workflow-hardening` closed a Scorecard Dangerous-Workflow finding
by refusing any `workflow_run` ref that is not a release tag. In the same cycle
`pose-package-channel-delivery` gave `package-channels.yml` the `workflow_run`
trigger it needed to fire at all — and, with it, a checkout of
`github.event.workflow_run.head_branch`. The finding did not move; it relocated.

That is confirmed rather than inferred: the Scorecard run at 2026-08-07T21:29Z
reports `score is 0: untrusted code checkout` against
`package-channels.yml:32`, where earlier runs reported it against
`verify-release.yml`. Token-Permissions, the other finding that cycle closed,
now has zero alerts in any state — so one of the two fixes held and the other
was undone by a sibling spec on the same day.

Reviewing each diff on its own merits is what let it through. Neither change
was wrong in isolation; the pair was.

While confirming the fix, `verify-release.yml` turned out to still interpolate
the same raw event value directly into a `run:` script — the hardening had
confined the checkout and left the script injection. Same class, same event
value, one line away from the fix that was already there.

### Constraints
- The clean-host matrix runs on macOS and Windows runners, so the resolution
  cannot assume a Linux shell by default; each step that touches the tag
  declares `shell: bash`.
- The tag must be validated once and consumed as data. Passing it forward as
  template text is the defect, not just checking it out.

### Non-goals
- Revisiting the `workflow_run` trigger itself. It is the trigger that fires;
  the problem was never that it fires, it was what the job did with its
  payload.

---

## 2. Requirements

### Functional
- R1: `package-channels.yml` shall not check out an event-supplied ref; it
  shall resolve the tag against the release pattern and refuse anything else
  before executing repository content.
- R2: No workflow shall interpolate an event-supplied ref into a `run:` script;
  the validated tag shall reach later steps through the environment.
- R3: The Scorecard Dangerous-Workflow finding shall be absent from the run
  after this change.

### Non-functional
- The gate keeps its behaviour on both runners; only the source of the tag
  changes.

### Security
- The job holds only `contents: read` and no secrets, which bounded the impact
  but never made the pattern sound. Two attacker-influenceable paths are closed:
  executing an arbitrary ref's content, and injecting into a shell script.

### Compatibility
- `workflow_dispatch` with an explicit `tag` input, and the `release:
  published` path, resolve through the same validation.

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/package-channels.yml` — resolution, checkout, and the
  three steps that consumed the raw ref.
- `.github/workflows/verify-release.yml` — the leftover interpolation.

### Artifacts
- created: .pose/specs/pose-package-channel-workflow-safety/spec.md
- modified: .github/workflows/package-channels.yml
- modified: .github/workflows/verify-release.yml

### Delivery targets
- governance:package-channel-workflow-safety module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- Nothing prevents a third workflow from acquiring a `workflow_run` trigger and
  the same pattern. The recurrence here was not a knowledge gap — the fix
  existed, in the repository, on the same day — so tooling is the only thing
  that would have caught it. Recorded as a follow-up rather than solved here.

---

## 4. Tasks

### Planning
- [x] Confirm the finding moved rather than closed, from the Scorecard run
- [x] Audit every workflow for the same interpolation

### Implementation
- [x] R1: resolve and confine the tag in package-channels.yml
- [x] R2: pass the validated tag by environment, in both workflows

### Validation
- [x] Workflow security contract and YAML parse
- [x] R3: confirm the finding is absent from a real Scorecard run

---

## 6. Validation

### Strategy
The claim is about a finding produced by a scanner this repository does not
run locally, so it splits in two. What is checkable locally: no workflow
interpolates an event-supplied ref into a script, the pinning and permissions
contract still passes, and every workflow still parses. What only the run can
settle: whether Scorecard agrees the pattern is gone.

Grep is the honest instrument for R2 — the property is the absence of a
substring across eight files, and asserting it any other way would be theatre.

### Deterministic checks

#### Security / Contract
- Command: `go -C pose-mcp test ./internal/version/... -run WorkflowSecurity -count=1`
- Scope: pinning and top-level permissions across every workflow
- Expected: pass

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, development build `pose 0.20.3-dev`.
- Notes: `grep -n 'run:.*github\.event\.' .github/workflows/*.yml` returns
  nothing after the change and returned `verify-release.yml:51` before it. All
  eight workflows parse as YAML. The contract test passes.

### Results summary
- Successes: no event-supplied ref reaches a checkout or a script
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:package-channel-workflow-safety evidence:integration check:delivery-integration report:.github/workflows/package-channels.yml — the job resolves the tag against the release pattern, refuses anything else, and checks out the validated tag by fetching it
- R2 [satisfied] check:workflow-ref-interpolation — no `run:` script in any workflow interpolates an event value; the validated tag travels as `RELEASE_TAG`
- R3 [satisfied] report:.github/workflows/package-channels.yml — the Scorecard run after this change reports no Dangerous-Workflow alert, where the run immediately before it reported score 0 against package-channels.yml:32

### Known gaps
- The `if:` guard still reads `head_branch` directly, which is correct — a
  condition is not an execution context — but means the raw value appears in
  two places for two different reasons, which reads as inconsistent.
- Nothing stops the next workflow from repeating this.

---

## 7. Final Report

### Delivered scope
`package-channels.yml` resolves the release tag once, refuses anything that is
not one, and passes it forward as an environment variable; `verify-release.yml`
stops interpolating the raw event value into its verification script.

### Files and modules changed
- .github/workflows/package-channels.yml
- .github/workflows/verify-release.yml

### Validation executed
- Command: workflow security contract, YAML parse, interpolation grep
- Result: pass; Dangerous-Workflow absent from the following Scorecard run

### Residual risks
- The pattern is prevented by knowledge, not by a check. It recurred once
  already, within a day of being fixed.

### Follow-ups

- [covered: pose-workflow-event-ref-contract] A check should fail when a workflow triggered by `workflow_run` or `pull_request_target` checks out an event-supplied ref or interpolates one into a `run:` script. This recurred within a day of being fixed, by a sibling spec, with the correct pattern already in the repository — review caught it only because Scorecard was consulted afterwards. `TestWorkflowSecurityContract` already parses every workflow and is the natural home. (owner:@pose-maintainers crit:medium review:2026-09-18)
