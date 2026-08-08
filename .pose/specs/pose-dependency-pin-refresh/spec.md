---
slug: pose-dependency-pin-refresh
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-dependency-digest-pinning
priority: 1
components: release
delivers: governance:dependency-pin-refresh
---

# Spec: Immutable pins get a way back up to date

## 1. Intent

### Goal
Give every pinned dependency an automated refresh path, and make the manifest
edit that a bump forces into a one-command step.

### Business value
`pose-dependency-digest-pinning` closed 84 Scorecard findings by making every
action, container base and Python package immutable. It also, deliberately,
traded one risk for another: a digest cannot be repointed at attacker-controlled
code, but an upstream security fix now arrives only when someone updates a SHA
by hand, and nothing reports that a pin has fallen behind. That spec recorded
the trade as its main residual risk and left the mechanism open.

The follow-up offered two options: Dependabot, or a check comparing each pin
against its tag's current target. The second only moves the manual work — it
tells a human to go and edit a SHA, which is what already was not happening.
Dependabot opens the pull request with the new digest and the updated version
comment, so the refresh happens rather than being reported.

Verified rather than assumed: no Dependabot configuration existed. 15 actions,
4 container `FROM` lines and 29 hash-pinned Python packages had no refresh path
of any kind.

### Constraints
- The runtime-currency gate requires each action's record to be pinned at the
  same ref, so a Dependabot PR fails CI until
  `.github/action-runtimes.json` is refreshed. That friction is the gate working
  as designed; what is not acceptable is leaving it as a manual chore on every
  PR.
- Monthly, grouped updates. Per-action PRs would each require the same manifest
  refresh, multiplying the friction by the number of actions.

### Non-goals
- Auto-merging. A major bump can carry behaviour changes — `setup-go@v6`'s
  toolchain handling and `checkout@v7`'s fork-PR block both did — so the PR is
  opened, not landed.
- Refreshing the deprecated-runtime list, which is human input with owners and
  announcement dates.

---

## 2. Requirements

### Functional
- R1: Dependabot shall cover every pinned ecosystem: GitHub Actions, the two
  Dockerfiles, the docs Python lock and the Go module.
- R2: Updates shall be grouped per ecosystem, so one cycle produces one
  manifest refresh rather than one per dependency.
- R3: A script shall regenerate `.github/action-runtimes.json` from the
  workflows as they stand, preserving the declared deprecated-runtime list.
- R4: That script shall be idempotent: run against an already-correct manifest
  it shall produce no change.

### Non-functional
- The refresh script makes one API call per distinct action, the same order as
  the online verifier.

### Security
- Dependabot raises PRs; it does not merge. The pinning contract still requires
  a full SHA, and a Dependabot bump preserves that form.
- The refresh script reads public `action.yml` contents and writes only the
  manifest.

### Compatibility
- No product change.

---

## 3. Technical Plan

### Affected areas
- `.github/dependabot.yml` — new.
- `scripts/refresh-action-runtimes.sh` — new.

### Artifacts
- created: .pose/specs/pose-dependency-pin-refresh/spec.md
- created: .github/dependabot.yml
- created: scripts/refresh-action-runtimes.sh

### Delivery targets
- governance:dependency-pin-refresh module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **The configuration was unverifiable at authoring time.** YAML validity was
  checkable here; whether Dependabot accepted every field — `directories` for the
  two Dockerfiles in particular — was only visible once the file reached the
  default branch. It did, minutes later, and all three grouped PRs opened. The
  risk is retained in this record because it was real when the decision was
  made, not because it remains open.
- Grouped monthly updates mean a security fix published the day after a run
  waits up to a month. The alternative — daily, ungrouped — produces a manifest
  refresh per PR, which is the friction this spec exists to bound. The trade is
  deliberate and is the reason the refresh is one command.

---

## 4. Tasks

### Planning
- [x] Establish that no refresh path existed and count what is pinned
- [x] Choose between Dependabot and a staleness check, with the reason

### Implementation
- [x] R1: cover the four ecosystems
- [x] R2: group per ecosystem, monthly
- [x] R3: regenerate the manifest, preserving the deprecated list
- [x] R4: idempotent regeneration

### Validation
- [x] Regenerate against the current manifest and diff
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
The half that is decidable here is the refresh script: run against a manifest
already known to be correct, it must produce exactly that manifest, and the
runtime gate must still pass afterwards. Idempotence is the property that
matters — a regenerator that rewrites its input differently each run would make
every Dependabot PR carry an unreviewable diff.

The Dependabot configuration cannot be validated locally beyond YAML syntax, so
it was pushed and observed rather than asserted — and the observation turned out
to be the strongest evidence in this spec: a real bump PR exercised the gate, the
message and the remedy in sequence.

### Deterministic checks

#### Test
- Command: `bash scripts/refresh-action-runtimes.sh` then diff against the previous manifest
- Scope: regeneration fidelity and idempotence
- Expected: no change; the runtime gate still passes

#### Security / Contract
- Command: `shellcheck --severity=warning ... scripts/*.sh`
- Scope: the new script
- Expected: no findings

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: the script resolved all 15 actions and produced a manifest byte-identical
  to the reviewed one, with the deprecated-runtime list preserved;
  `TestActionRuntimeCurrency` passes against the regenerated file. The
  configuration parses as YAML and declares four ecosystems.
- **Settled by the service within minutes of the push**, so the risk this spec
  recorded as unverifiable was verified after all: Dependabot accepted the
  configuration and opened three grouped PRs — go modules (#11), container
  images across both directories (#10, confirming `directories` is valid) and
  actions (#12). The go PR passed CI unchanged. The actions PR failed exactly as
  designed, with `ossf/scorecard-action: pinned at 2d1146689b8c but recorded at
  4eaacf0543bb`. Running `scripts/refresh-action-runtimes.sh` on that branch
  produced a one-line diff and CI went green (run 31281336580, after 31281261083
  failed). The friction, the message and the remedy all behaved as specified.

### Results summary
- Successes: every pinned ecosystem has a refresh path; the manifest edit a
  bump forces is one command and provably idempotent; the whole loop —
  configuration accepted, PRs opened, gate fires, one-command remedy, CI green —
  demonstrated end to end on a real Dependabot PR
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:dependency-pin-refresh evidence:integration check:delivery-integration report:.github/dependabot.yml — github-actions, docker (both Dockerfiles), pip (the hash-pinned docs lock) and gomod are declared; the service accepted all four and opened PRs #10, #11 and #12
- R2 [satisfied] report:.github/dependabot.yml — each ecosystem declares a group matching `*` on a monthly interval
- R3 [satisfied] test:scripts/refresh-action-runtimes.sh — the script rebuilds `runtimes` from the workflows and carries `deprecated_runtimes` and the schema comment across unchanged
- R4 [satisfied] check:action-runtime-currency — regenerating the reviewed manifest produced no diff, and the offline gate passes against the result; on PR #12 the same command produced exactly the one-line diff the bump required

### Known gaps
- A monthly cadence bounds how fast a published fix can arrive.
- Nothing refreshes the manifest on the PR automatically; the gate fails it,
  which is the signal, but the fix is still a human step. Demonstrated to be a
  one-command step, not a small one that stays undone.

---

## 7. Final Report

### Delivered scope
Every pinned ecosystem has an automated refresh path, and the manifest edit a
bump forces is a single idempotent command.

### Files and modules changed
- .github/dependabot.yml
- scripts/refresh-action-runtimes.sh

### Validation executed
- Command: regeneration + diff; `TestActionRuntimeCurrency`; shellcheck
- Result: identical manifest, gate green, no findings

### Residual risks
- Monthly grouping is a deliberate ceiling on how fast a published fix arrives.

### Follow-ups

- [done] Confirmed within minutes of the push: Dependabot accepted the configuration and opened PRs #10 (images across both directories, validating `directories`), #11 (go modules) and #12 (actions). Original item: confirm on the default branch that Dependabot accepted the configuration and that the first grouped run opens the PRs it should. (owner:@pose-maintainers crit:medium review:2026-09-18)
- [open] A Dependabot PR fails CI until `.github/action-runtimes.json` is refreshed, and nothing automates that step on the PR branch — observed on PR #12, fixed by one command. Consider a workflow that runs `scripts/refresh-action-runtimes.sh` and commits the result when the actor is Dependabot, which would turn the gate's deliberate friction into a self-healing one. Worth weighing against the argument for keeping a human in the loop on every dependency bump. (owner:@pose-maintainers crit:low review:2026-11-06)
