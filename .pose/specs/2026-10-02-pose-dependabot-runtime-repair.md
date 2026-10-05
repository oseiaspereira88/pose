---
slug: pose-dependabot-runtime-repair
status: done
created_at: 2026-10-02
completed_at: 2026-10-02
priority: 1
components: pose-mcp
task_type: feature
delivers: governance:dependabot-runtime-repair
---
# Spec: Repair the runtime manifest for bounded Dependabot pin updates

## 1. Intent
### Goal
Prevent repeated manual repairs after GitHub Actions dependency bumps.
### Business value
PR preparation for 6.2.0 encountered four stale runtime refs after automated upstream bumps.
### Constraints
Run trusted default-branch code only. Read PR files as data. Write only the runtime manifest, never approve or merge the PR. Preserve security gates.
### Non-goals
Dependency selection, arbitrary PR repair, fork writes, automatic merge.

## 2. Requirements
### Functional
- R1: Inspect only an open same-repository Dependabot GitHub Actions PR at the exact tested revision after CI completes.
- R2: Accept only existing workflow modifications consisting of full SHA pins and version comments, plus the derived runtime manifest from a prior repair; reject other files, workflow logic, new action identities, mutable refs and conflicting pins.
- R3: Resolve runtimes from each pinned action and retain trusted deprecated-runtime policy; write only the manifest through a non-forced commit with the tested head as parent.
- R4: Make an unchanged manifest a no-op and dispatch CI after repair; a concurrent branch change must reject the write.
### Security
No PR checkout or execution, no caches or extra secrets. The repair job alone receives contents/actions write permission.
### Compatibility
Existing offline and online runtime gates stay mandatory.

## 3. Technical Plan
### Artifacts
- created: scripts/repair-dependabot-runtimes.py
- created: .pose/adr/2026-10-02-trusted-dependabot-runtime-repairs.md
- created: tests/release/repair-dependabot-runtimes.py
- created: .github/workflows/repair-dependabot-runtimes.yml
- modified: .github/workflows/ci.yml
- modified: pose-mcp/internal/version/contract_test.go
- created: .pose/changelogs/unreleased/pose-dependabot-runtime-repair.md
### Delivery targets
- governance:dependabot-runtime-repair module:pose-mcp profile:release-governance entrypoint:.github/workflows/repair-dependabot-runtimes.yml
### API/contract changes
Git-data writes create one child commit and update the branch without force; CI gains workflow_dispatch. CLI contracts unchanged.
### Technical risks
Validate identity, SHA-only changes and parent revision. Token-driven pushes do not trigger CI, so dispatch explicitly.

## 4. Tasks
### Implementation
- [x] Add bounded trusted repair and read-only PR inspection.
- [x] Preserve no-op, concurrency rejection and explicit CI dispatch.
### Validation
- [x] Run provider-boundary positive/negative tests and workflow security contracts.

## 5. Decisions
### Decision 1
- Date: 2026-10-02
- Decision: use workflow_run and Git-data API writes with trusted scripts; never check out PR code under write authority.
- Rationale: only the derived runtime manifest needs automation; the PR keeps normal CI and maintainer review.
- ADR: .pose/adr/2026-10-02-trusted-dependabot-runtime-repairs.md
- References: https://docs.github.com/en/actions/reference/security/securely-using-pull_request_target ; https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow

## 6. Validation
### Strategy
Apply pose-test-plan. Mock provider calls at the boundary; assert the complete mutation sequence and dispatch, plus no writes for unauthorized, mixed, stale and no-op inputs. Run the generator against current pins without changing policy.
### Deterministic checks
| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Positive, hostile, stale and no-op cases (required) | python3 tests/release/repair-dependabot-runtimes.py | Only the runtime JSON changes; invalid input never updates a branch |
| Workflow authority and pins (required) | go -C pose-mcp test ./internal/version -count=1 | Security and runtime contracts pass |
| Module regression (required) | pose validate --strict --module pose-mcp --json-out .pose/results/delivery-validation.json --report | Canonical matrix passes |
### Execution log
Seven provider-boundary tests, workflow authority contracts and the canonical module matrix passed on 2026-10-02 (50/50 checks). ShellCheck 0.11.0 also passed the complete CI script set.
### Requirement trace
- R1 [satisfied] check:dependabot-runtime-repair-integration — exact tested head, same repository, bot identity and open state are checked before generation.
- R2 [satisfied] check:dependabot-runtime-repair-integration — mixed changes, action-identity edits, mutable refs and conflicts are refused.
- R3 [satisfied] check:dependabot-runtime-repair-integration — trusted policy and a one-path Git-data commit are asserted; branch writes are non-forced.
- R4 [satisfied] check:dependabot-runtime-repair-integration — no-op, stale-head rejection and explicit workflow dispatch are covered.
### Known gaps
First real Dependabot repair requires the workflow on the default branch; tests prove the API boundary before deployment.

## 7. Final Report
### Delivered scope
A trusted default-branch workflow repairs only derived runtime evidence for eligible Dependabot action pins. PR review and existing security gates remain required.
### Coverage of earlier follow-ups

This spec covers two earlier follow-ups: the automation asked for in pose-dependency-pin-refresh (a Dependabot PR fails CI until `.github/action-runtimes.json` is refreshed) and the same defect recorded in pose-release-security-gate-integrity (every actions bump breaks `main` until fixed by hand). Its tests run in `go test ./...` through `TestDependabotRuntimeRepairTrustBoundary`. Confirmed by the maintainer on 2026-10-05 while resolving the backlog reconciliation (spec pose-open-backlog-reconciliation, Decision 5). It had not yet processed a real Dependabot PR at that date: the first one after the workflow landed is the end-to-end proof.

### Follow-ups
None.
