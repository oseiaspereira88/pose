---
slug: pose-release-workflow-hardening
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-ossf-security-baseline
priority: 1
components: release
delivers: governance:release-workflow-hardening
---

# Spec: Least privilege and a confined checkout before going public

## 1. Intent

### Goal
Close the two workflow findings the OpenSSF baseline reported: an
event-supplied checkout ref in `verify-release.yml` and repository-wide
`contents: write` in `release.yml`.

### Business value
Both were recorded as findings when the baseline was triaged and both were
correctly judged low-impact for a private repository: the verify job holds only
read scopes and no secrets, and the release job genuinely needs write.

Opening the repository changes the calculus. `workflow_run` runs in the base
repository's context, so a ref an outsider can influence is a different risk
once outsiders exist, and a top-level write scope applies to every job that may
later be added to the file.

### Constraints
- The release must keep working: the publish step genuinely needs
  `contents: write`, `id-token: write` and `attestations: write`.

### Non-goals
- Changing what the verification does, only which ref it is allowed to run.

---

## 2. Requirements

### Functional
- R1: `verify-release.yml` shall verify only refs that match the release tag
  pattern, and shall refuse anything else with a named error.
- R2: `release.yml` shall grant `contents: read` at the top level and raise the
  write scopes on the job that publishes.

### Non-functional
- No additional latency beyond one tag fetch.

### Security
- The refusal must happen before any repository content is executed.

### Compatibility
- No product change; both workflows keep their published behaviour for real
  release tags.

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/verify-release.yml` — resolved checkout.
- `.github/workflows/release.yml` — scoped permissions.

### Artifacts
- created: .pose/specs/pose-release-workflow-hardening/spec.md
- modified: .github/workflows/verify-release.yml
- modified: .github/workflows/release.yml

### Delivery targets
- governance:release-workflow-hardening module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- The tag-pattern guard rejects a ref that is not `vN.N.N`, which is the intent;
  a future pre-release convention would need the pattern widened deliberately.

---

## 4. Tasks

### Planning
- [x] Re-read the Scorecard findings against the public-release context

### Implementation
- [x] R1: check out the default branch, then resolve and check out the tag
- [x] R2: contents: read at top level, write on the release job

### Validation
- [x] Workflow contract test passes
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
The workflow-contract test already enforces permission and pinning invariants,
so it is the local gate. The refusal path is only observable in a real
`workflow_run`, so it is asserted by construction and reviewed rather than
executed here.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/version -run 'WorkflowSecurity' -count=1`
- Scope: workflow permissions and action pinning
- Expected: ok

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.20.3-dev.
- Notes: the OpenSSF baseline of 2026-08-07 reported Dangerous-Workflow 0 for
  the checkout and Token-Permissions 0 for the top-level write scope.

### Results summary
- Successes: workflow contract test green
- Failures: none
- Warnings: the refusal path executes only on a real workflow_run

### Requirement trace
- R1 [satisfied] governance:release-workflow-hardening evidence:integration check:delivery-integration report:.github/workflows/verify-release.yml — the job checks out the default branch, then accepts only a `vN.N.N` tag, fetching and checking it out explicitly; anything else exits with the ref named
- R2 [satisfied] check:workflow-security test:TestWorkflowSecurity — the workflow grants `contents: read` at the top level, with the write scopes declared on the release job

### Known gaps
- The refusal branch has not executed; it is reachable only from a real
  `workflow_run` with a non-tag ref.

---

## 7. Final Report

### Delivered scope
`verify-release.yml` resolves and validates the tag it verifies instead of
checking out an event-supplied ref, and `release.yml` grants least privilege at
the top level with the write scopes on the publishing job.

### Files and modules changed
- .github/workflows/verify-release.yml
- .github/workflows/release.yml

### Validation executed
- Command: `go -C pose-mcp test ./internal/version -run 'WorkflowSecurity'`
- Result: ok

### Residual risks
- Scorecard recomputes on a schedule, so the score improvement is not
  observable in this change; it should be re-read after the next run.

### Follow-ups

- [open] Re-read the OpenSSF score after the next Scorecard run and confirm Dangerous-Workflow and Token-Permissions moved off zero. (owner:@pose-maintainers crit:medium review:2026-09-18)
- [open] Pinned-Dependencies is 2: GitHub-owned actions are still referenced by tag rather than digest. Decide whether to pin them before the repository goes public. (owner:@pose-maintainers crit:medium review:2026-09-18)
