---
slug: pose-review-root-binding-parity
status: done
created_at: 2026-09-25
completed_at: 2026-09-26
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:review-root-binding-parity
---

# Spec: Review parity for root bindings and delivery preconditions

## 1. Intent

### Goal
Make review planning and attestation honor valid root delivery targets, classify the root MCP binding, and distinguish required validation evidence from a declared delivery target.

### Business value
An adopting repository can review its root MCP configuration without misclassifying the subject or claiming that a global check passed when unrelated modules lack environment prerequisites.

### Constraints
- Keep unclassified files and escaping paths fail closed.
- Keep validation evidence required for specs with implementation components.
- Preserve sealed bundles and existing review attestations.
- `knowledge:adr-sealed-review-bundles-review` requires input provenance and subject classification to remain explicit.

### Non-goals
- Change the adopted repository's review policy, skip an applicable module check, or alter historical attestations.

## 2. Requirements

### Functional
- R1: A delivery target with `module:.` must not add an invalid component-path blocker; traversal and symlink escapes still block.
- R2: An attributed `.mcp.json` binding must be classified as governed configuration and included in the immutable review subject.
- R3: A spec with an implementation component but no delivery target must require current validation evidence while allowing a `delivery-target-declared` review tool to be deferred with an explicit reason.

### Security and compatibility
- Do not infer a root component for arbitrary files or weaken the unclassified-path blocker.
- Keep the attestation contract and previously sealed gate semantics compatible.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose` review plan, bundle classification, attestation coverage and focused regression tests.

### Artifacts
- created: .pose/specs/2026-09-25-pose-review-root-binding-parity.md
- created: .pose/adr/2026-09-25-review-evidence-versus-delivery-target.md
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_plan_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go

### Delivery targets
- contract:review-root-binding-parity module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Contract change
- Review plans accept root module targets and include root MCP configuration; review tool preconditions use actual delivery declarations.

### Risks and rollback
- A false component-root match could hide an unsafe path. Limit the exception to the delivery target's exact `module:.` field and retain path validation elsewhere.
- Revert this change set and the adopting gitlink together if the new classification misattributes a subject.

## 4. Tasks

- [x] Reproduce the three blockers in a pinned adopter and inspect existing root-manifest and doc-only review contracts.
- [x] Run `pose assess discover --component pose-mcp` before code changes (49,904 production LOC, 37,713 test LOC).
- [x] Add focused negative and positive regression tests.
- [x] Apply the bounded implementation and validate the Go module.
- [ ] Review and close in a separate execution.

## 5. Decisions

- D1: Keep validation evidence and delivery-target presence as separate facts, per the accepted ADR for this spec.
- D2: Treat `.mcp.json` as an exact governed manifest rather than broadening classification of all dotfiles.

## 6. Validation

### Test plan before implementation

| Scenario | Required command | Expected evidence |
| --- | --- | --- |
| Root target and escapes | `go test ./internal/pose -run 'TestReviewPlanRootDeliveryTarget|TestReviewPlanRejectsComponentAndArtifactSymlinkEscapes' -count=1` from `pose-mcp` | `module:.` has no blocker; unsafe paths remain blocked. |
| MCP binding classification | `go test ./internal/pose -run TestReviewBundleClassifiesRootManifestsAndProjectFiles -count=1` from `pose-mcp` | `.mcp.json` is included as governance; unknown files still block in the existing negative test. |
| Component without target | `go test ./internal/pose -run TestReviewValidationPreconditionForComponentWithoutTarget -count=1` from `pose-mcp` | Evidence remains required; target-gated tool may be deferred. |
| Module regression | `go test ./...` and `go vet ./...` from `pose-mcp` | Exit 0. |
| Registered contract | `pose validate --strict --module pose-mcp` from project root | All required module checks pass. |

### Execution log
- 2026-09-25: adopter bundle `rvb-7c583c0ba24782c0` reports invalid `module:.` and unclassified `.mcp.json`; a separate adopter attestation reports invalid deferred `validate` for a componentful spec with no delivery target.
- 2026-09-25: all three focused regressions failed before the implementation and passed after it; the root-target test now requires zero plan blockers and the target-precondition test checks both absent and declared delivery targets.
- 2026-09-25: `go test ./...`, `go vet ./...`, and `go build ./cmd/pose` passed. `pose validate --strict --module pose-mcp` passed 25/25 checks with a writable Go cache and local sockets enabled. The first sandboxed attempts failed on a read-only cache and forbidden local sockets, independently reproduced by the observability test.
- 2026-09-25: `pose lint-spec pose-review-root-binding-parity --ready-check` passed. `pose assess integrate` reported 57 contracts and 56 existing unobserved-consumer inventory gaps; none is attributed to this review change. `pose docs-check` is unavailable because this source repository has no docs manifest.
- 2026-09-25: the corrected local binary prepared the adopter review bundle without the prior root-path and `.mcp.json` blockers; the remaining blocker was its uncommitted `pose-dist` gitlink. It verified the existing harness attestation as fresh and approved.
- 2026-09-25: a separate review corrected the requirement trace disposition to `satisfied`; strict lint and all four focused tests passed. The full `pose-mcp` matrix passed 25/25 outside the sandbox after socket-bound MCP tests failed under sandbox restrictions. `go vet ./...` passed. `assess tech-debt` found no markers; `assess integrate` retained 56 previously inventoried consumer gaps among 57 contracts.

### Requirement trace
- R1 [satisfied] test:TestReviewPlanRootDeliveryTarget
- R2 [satisfied] test:TestReviewBundleClassifiesRootManifestsAndProjectFiles
- R3 [satisfied] test:TestReviewValidationPreconditionForComponentWithoutTarget

### Known gaps
- Governed review and adopter re-pin are pending.

## 7. Final Report

### Delivered scope
- Review planning accepts root-module targets, root MCP configuration is governed subject input, and attestation tool preconditions follow sealed delivery declarations.

### Validation executed
- Source module tests, vet, build, and strict POSE matrix passed; adopter read-only probes confirmed the three original blockers were removed.

### Residual risks
- The adopter remains on the prior pin until this source change is reviewed. A root delivery target still needs a concrete entrypoint, which is validated by the existing target parser.

### Follow-ups
- [open] Re-pin and validate the adopting repository only after this source spec passes review and closeout. (owner:@pose-maintainers crit:high review:2026-10-02)
