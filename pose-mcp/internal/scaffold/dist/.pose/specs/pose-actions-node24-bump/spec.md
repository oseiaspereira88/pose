---
slug: pose-actions-node24-bump
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-ossf-security-baseline
priority: 1
components: release
delivers: governance:actions-runtime-currency
---

# Spec: The workflows run on the runtime they declare

## 1. Intent

### Goal
Move every first-party action off the deprecated Node.js 20 runtime, so the
workflows stop depending on a substitution the runner performs for them.

### Business value
Every CI run since 2026-08-07 carries the annotation that `actions/checkout@v4`,
`actions/setup-go@v5` and `actions/upload-artifact@v4` target Node.js 20 and are
being forced onto Node.js 24. The workflows pass, but they pass because the
runner overrides what they declare. That override is a transition measure and
its removal date belongs to GitHub, not to this project: when it goes, every
workflow — CI, release, verification, the security scanners — fails at once, on
a schedule nobody here controls.

Verified rather than assumed: each major's `action.yml` was read at its tag and
the `runs.using` value recorded, rather than inferring support from version
numbers.

| action | before | `using` | after | `using` |
| --- | --- | --- | --- | --- |
| actions/checkout | v4 | node20 | v7 | node24 |
| actions/setup-go | v5 | node20 | v7 | node24 |
| actions/upload-artifact | v4 | node20 | v7 | node24 |
| actions/setup-python | v5 | node20 | v7 | node24 |
| actions/deploy-pages | v4 | node20 | v5 | node24 |
| actions/dependency-review-action | v4 | node20 | v5 | node24 |
| github/codeql-action/* | v3 | node20 | v4 | node24 |

`actions/upload-pages-artifact` and `actions/attest-build-provenance` are
composite actions and declare no Node runtime; the former is moved to v5 for
currency, the latter is already on its newest major.

### Constraints
- Third-party actions stay pinned to full commit SHAs. This change touches only
  the `actions/` and `github/` orgs, which the
  `first-party-actions-tag-pinning` exception permits to be tag-pinned.
- Each major is taken to its newest release rather than to the lowest one that
  declares node24: the lowest merely repeats this work at the next deprecation.

### Non-goals
- Re-litigating the tag-pinning exception, which expires 2026-10-19 and is
  reviewed on its own schedule.
- Bumping the SHA-pinned third-party actions, a separate axis with a separate
  verification cost.

---

## 2. Requirements

### Functional
- R1: No workflow shall reference an action whose `action.yml` declares
  `runs.using: node20`.
- R2: The workflows shall keep working: CI, security, scorecard and the
  governance audit shall pass on the bumped majors.
- R3: The Node.js 20 deprecation annotation shall be absent from the run.

### Non-functional
- No change to what any workflow does; only the runtime the actions execute on.

### Security
- The pinning contract is unchanged and still enforced by
  `TestWorkflowSecurityContract`: first-party by tag, third-party by SHA.
- `actions/checkout@v7` additionally blocks checking out a fork PR under
  `pull_request_target` and `workflow_run` — the same class of exposure
  `pose-release-workflow-hardening` closed by hand in `verify-release.yml`.
  This project's `workflow_run` triggers fire from tag pushes on the base
  repository, not from fork PRs, so the new block does not reach them.

### Compatibility
- `actions/setup-go@v6` changed toolchain selection. The workflows pin
  `go-version: "1.26.5"` explicitly, which is the input that change makes more
  consistent rather than less.
- node24 majors require Actions Runner >= 2.327.1; GitHub-hosted runners are
  well past it.

---

## 3. Technical Plan

### Affected areas
- All eight workflows under `.github/workflows/`.

### Artifacts
- created: .pose/specs/pose-actions-node24-bump/spec.md
- modified: .github/workflows/ci.yml
- modified: .github/workflows/security.yml
- modified: .github/workflows/scorecard.yml
- modified: .github/workflows/release.yml
- modified: .github/workflows/verify-release.yml
- modified: .github/workflows/package-channels.yml
- modified: .github/workflows/governance-audit.yml
- modified: .github/workflows/docs.yml
- modified: .pose/specs/pose-ossf-security-baseline/spec.md

### Delivery targets
- governance:actions-runtime-currency module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- `actions/upload-artifact@v7` moved to ESM and added a direct-upload mode. The
  default behaviour is unchanged and no workflow sets the new `archive` input,
  but the artifact steps are the least exercised of the bumped set — only the
  release and governance-audit paths upload.
- Nothing enforces runtime currency going forward: the next deprecation is
  found the same way this one was, by reading an annotation on a passing run.

---

## 4. Tasks

### Planning
- [x] Read `runs.using` at each candidate major instead of inferring it
- [x] Confirm the pinning contract permits tag pins for these orgs

### Implementation
- [x] R1: bump every first-party action off node20
- [x] R2: keep the pinning contract green

### Validation
- [x] Workflow security contract
- [x] Run the mandatory checks
- [x] R3: confirm the annotation is gone from a real run

---

## 6. Validation

### Strategy
The runtime claim is not verifiable locally — no local check reads
`runs.using` for a referenced action. What is verifiable locally is that the
pinning contract still holds and the repository's own gates pass; the runtime
claim is settled by the run itself, which is where the annotation appears and
therefore where its absence is evidence.

### Deterministic checks

#### Security / Contract
- Command: `go -C pose-mcp test ./internal/version/... -run WorkflowSecurity -count=1`
- Scope: pinning contract and top-level permissions across every workflow
- Expected: pass

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64; majors resolved against the GitHub API at their tags.
- Notes: `TestWorkflowSecurityContract` passes on the bumped set. The CI,
  Security, Scorecard and Governance audit runs for this change completed
  successfully with no Node.js 20 annotation, where the immediately preceding
  runs carried it.

### Results summary
- Successes: every first-party action on node24; pinning contract green;
  annotation absent from the run
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:actions-runtime-currency evidence:integration check:delivery-integration report:.github/workflows/ci.yml — no workflow references a node20 action; each replacement's `runs.using` was read at its tag
- R2 [satisfied] check:workflow-security-contract — the pinning and permissions contract passes, and the four workflows the change touches completed successfully on the bumped majors
- R3 [satisfied] report:.github/workflows/ci.yml — the deprecation annotation that accompanied every prior run is absent

### Known gaps
- The composite actions (`upload-pages-artifact`, `attest-build-provenance`)
  declare no runtime; whatever they invoke internally is outside this trace.
- Runtime currency has no standing gate. The next deprecation surfaces as an
  annotation on a passing run, exactly as this one did.

---

## 7. Final Report

### Delivered scope
Every first-party action in the eight workflows moved to a major that declares
Node.js 24. The pinning contract is unchanged and still enforced; third-party
actions remain SHA-pinned.

### Files and modules changed
- .github/workflows/*.yml (eight files)
- .pose/specs/pose-ossf-security-baseline/spec.md

### Validation executed
- Command: `go -C pose-mcp test ./internal/version/... -run WorkflowSecurity`
- Result: pass; and the real runs completed with the annotation gone

### Residual risks
- The artifact-upload paths are exercised only by release and governance-audit
  runs, so `upload-artifact@v7` gets less coverage on an ordinary push than the
  other bumps do.

### Follow-ups

- [open] Runtime currency is detected by reading annotations on passing runs, which is how the Node 20 deprecation went unnoticed until it was already being forced. Consider a check that fails when a referenced first-party action declares a runtime GitHub has deprecated — the data is in each action's `action.yml` at its pinned tag. (owner:@pose-maintainers crit:low review:2026-11-06)
