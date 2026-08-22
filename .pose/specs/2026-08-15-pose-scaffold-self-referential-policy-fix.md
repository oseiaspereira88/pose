---
slug: pose-scaffold-self-referential-policy-fix
status: done
created_at: 2026-08-15
completed_at: 2026-08-15
supersedes:
depends_on:
priority: 0
components: scaffold, cli, mcpserver
delivers: capability:pose-mcp
---

# Spec: pose-scaffold-self-referential-policy-fix

---

## 1. Intent

### Goal
Stop the embedded scaffold from shipping pose-mcp's own `.pose/policy/
{delivery,artifacts}.json` roots to installed instances, and close the
related stdio identity documentation gap — both reported in GitHub issue #17.

### Business value
Every `pose install`/`pose update --force` on a target project silently
emptied that project's delivery-integrity and artifact-contract graphs,
because the seeded `roots`/`governed_roots` pointed at pose-mcp's own source
tree instead of the target's. A finished spec's `review bundle --seal` failed
with "no immutable attributed change set exists" even though the project had
done nothing wrong — the contamination came from the installer. Fixing this
at the source removes a class of confusing, hard-to-diagnose failures for
every downstream adopter, and the `pose doctor` addition gives already-bitten
instances (installed on 1.2.0, before this fix) a way to find and repair it
without re-discovering the root cause from scratch.

### Constraints
- No public contract change: `delivery.json`/`artifacts.json` schemas are
  unchanged, only their *shipped default content* changes.
- Must not regress the scaffold drift guard
  (`TestEmbeddedDistMatchesPoseDist`), which enforces byte-identical sync
  between `pose-dist/` and the embedded copy.
- `pose install`/`pose update` must keep never overwriting an existing
  policy file (unrelated invariant, must stay intact).

### Non-goals
- Not a redesign of Execution Identity (ADR-007) for stdio — that gap
  (issue #17 achado 2) was already correctly documented in the MCP-conductor
  ADR and `docs-site/docs/mcp.md`; this spec only adds the missing
  cross-reference from the runtime error message and `mcp-enforce/README.md`
  to the already-existing local alternative (`pose review attest`/`auto-attest`).
- Not an automatic repair of instances already contaminated by 1.2.0 —
  `pose install`/`update` deliberately never overwrite an existing policy
  file; this spec only adds detection (`pose doctor`), not silent mutation.

---

## 2. Requirements

### Functional
- R1: The embedded scaffold shall never ship `.pose/policy/delivery.json`
  `roots` or `.pose/policy/artifacts.json` `governed_roots` containing
  literal paths into pose-mcp's own source tree.
- R2: A project that already has a contaminated `delivery.json`/
  `artifacts.json` (all configured roots absent from its own filesystem)
  shall be flagged by `pose doctor` with a hint naming the cause.
- R3: `pose_validate_approve` invoked over the stdio transport shall return
  a diagnostic explaining the structural (non-configuration) cause and
  naming `pose review attest`/`pose review auto-attest` as the supported
  local alternative.

### Non-functional
- The neutral placeholder policy content must remain schema-valid and
  degrade to a no-op (`enabled: false`, empty roots) rather than erroring on
  load, so a fresh install never breaks `pose check`/`pose index` before the
  project configures its own roots.

### Security
- No security-sensitive surface touched; the stdio diagnostic change does
  not alter the authorization decision (approval is still denied), only the
  message text.

### Compatibility
- `enabled: false` on the shipped default is a behavior change from the
  previous (contaminated but nominally `enabled: true`) default, but the
  previous default never resolved any real target-project root, so this
  changes documented/observable behavior for zero previously-working
  configurations.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/distpolicy` (new exclusion + placeholder source of truth)
- `pose-mcp/internal/scaffold/gen` (generator consumes the placeholder)
- `pose-mcp/internal/scaffold/dist/.pose/policy` (regenerated embedded output)
- `pose-mcp/internal/scaffold` (drift-guard test)
- `pose-mcp/internal/cli` (`pose doctor` detection)
- `pose-mcp/internal/mcpserver` (stdio-specific approval diagnostic)
- `mcp-enforce` (README cross-reference)

### Artifacts
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- created: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: pose-mcp/internal/scaffold/gen/main.go
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/delivery.json
- modified: pose-mcp/internal/scaffold/dist/.pose/policy/artifacts.json
- modified: pose-mcp/internal/scaffold/scaffold_test.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_remediation_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/validate_orchestration_test.go
- modified: mcp-enforce/README.md
- created: .pose/knowledge/2026-08-15-decision-log-self-referential-policy-template-contamination.md
- created: .pose/reports/2026-08-15-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
None. `pose_validate_approve`'s error *message* changes when denied over
stdio; the decision (deny) and error shape (JSON-RPC tool error,
`isError: true`) are unchanged.

### Data/storage changes
None.

### Technical risks
- The `pose doctor` heuristic (warn only when *every* configured root is
  absent) is conservative by design and will not flag a partially-contaminated
  policy file. Accepted: a false negative there still leaves the existing
  delivery-integrity checks as the backstop, and a false positive would train
  operators to ignore the hint.

---

## 4. Tasks

### Planning
- [x] Confirm intent (GitHub issue #17, both reported findings)
- [x] Identify affected modules (scaffold/distpolicy, scaffold/gen, cli/doctor, mcpserver)

### Implementation
- [x] Exclude `delivery.json`/`artifacts.json` from the wholesale `.pose/policy` sync
- [x] Ship an explicit neutral placeholder instead, from a single source of truth
- [x] Regenerate the embedded scaffold and adjust the drift guard
- [x] Add `pose doctor` detection for already-contaminated instances
- [x] Improve the stdio-specific `pose_validate_approve` diagnostic
- [x] Cross-reference the stdio gap in `mcp-enforce/README.md`

### Validation
- [x] Run the mandatory checks (below)

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: How to keep the generator and the drift-guard test from
  re-diverging on what the embedded scaffold contains for these two files
  (the exact failure mode that let the original contamination through
  unnoticed for two releases).
- Options considered:
  (a) Duplicate the placeholder JSON literal in both `gen/main.go` and
  `scaffold_test.go`; (b) put the placeholder behind a single exported
  function in the existing `distpolicy` package, which both already import
  for `IsIncluded`.
- Decision: (b).
- Rationale: `distpolicy` already exists specifically because "the generator
  and the drift test both used to carry their own copy" of the inclusion
  rules and drifted (see its package doc). The same failure mode applies
  one-for-one to placeholder content; reusing the same package closes it the
  same way.
- Consequences: `distpolicy` now owns both "what's included" and "what
  replaces what's excluded" for this one case — a slightly wider
  responsibility than its name suggests, but consistent with why it exists.

---

## 6. Validation

### Strategy
Full `go test`/`go vet`/`go build` on both affected modules (`pose-mcp`,
`mcp-enforce`), plus the repository's own deterministic POSE gates, before
declaring this spec done.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./...` and `go -C mcp-enforce test ./...`
- Scope: pose-mcp (all packages, including new `distpolicy_test.go` and
  extended `doctor_remediation_test.go`/`validate_orchestration_test.go`),
  mcp-enforce
- Expected: all packages pass, zero failures

#### Lint
- Command: `gofmt -l pose-mcp`
- Scope: pose-mcp module
- Expected: empty output (no formatting drift)

#### Typecheck
- Command: n/a (Go — covered by build)
- Scope: n/a
- Expected: n/a

#### Build
- Command: `go -C pose-mcp build ./...` and `go -C mcp-enforce build ./...`
- Scope: both modules
- Expected: clean build, zero errors

#### Security / Contract
- Command: `go -C pose-mcp vet ./...` and `go -C mcp-enforce vet ./...`;
  `pose validate --tolerant --module pose-mcp --report`; `pose validate
  --tolerant --module mcp-enforce --report`
- Scope: both modules, plus the repo's own delivery/surface/review-plan
  contract test subsets bundled into the `pose-mcp` validation-matrix entry
- Expected: clean vet, `pose validate` reports `Result: SUCCESS`

### Execution log
- Date: 2026-08-15
- Environment: linux/amd64, go1.26.5, local dev checkout of pose-dist (this repository)
- Notes: `go generate ./internal/scaffold` run after the distpolicy/gen
  change to regenerate the embedded `.pose/policy/{delivery,artifacts}.json`
  before running the drift-guard test.

### Results summary
- Successes: `go build ./...`, `go vet ./...`, `gofmt -l .` (empty), `go test
  ./...` — both `pose-mcp` and `mcp-enforce` — all green, including the new
  regression tests in `distpolicy_test.go`, `doctor_remediation_test.go` and
  `validate_orchestration_test.go`. `pose validate --tolerant --module
  pose-mcp --report` and `--module mcp-enforce --report` both report
  `Result: SUCCESS`.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] check:TestSelfReferentialPolicyFilesExcluded
  check:TestNeutralPolicyTemplatesAreSchemaValidAndInert
  check:TestEmbeddedDistMatchesPoseDist commit:06c5fa1
- R2 [satisfied] check:TestDoctorDetectsContaminatedPolicyRoots
  check:TestDoctorSilentOnLegitimatePolicyRoots commit:06c5fa1
- R3 [satisfied] check:TestApproveOverStdioNamesTheStructuralGapAndTheAlternative
  commit:06c5fa1

### Known gaps
- No automated migration for instances already seeded with the contaminated
  policy before this fix — `pose doctor` detects it, but the operator still
  edits `.pose/policy/{delivery,artifacts}.json` by hand. Deliberate scope
  boundary (see Non-goals), not a gap in this spec's own checks.

---

## 7. Final Report

### Delivered scope
Fixed the root cause of GitHub issue #17 achado 1 (scaffold policy
contamination) at the source, added detection for already-affected
instances, and closed the documentation/cross-reference gap from achado 2
(stdio + `pose_validate_approve`). Did not redesign Execution Identity for
stdio (out of scope — the mechanism gap was already intentional and
documented; only the diagnostic and README were missing the pointer to the
existing local alternative) and did not attempt to auto-repair already-
contaminated instances (an explicit `pose install`/`update` invariant this
spec must not violate).

### Files and modules changed
See Artifacts above.

### Validation executed
- Command: `go -C pose-mcp build ./... && go -C pose-mcp vet ./... && go -C pose-mcp test ./... && gofmt -l pose-mcp`
- Result: all green
- Command: `cd mcp-enforce && go build ./... && go vet ./... && go test ./...`
- Result: all green
- Command: `pose validate --tolerant --module pose-mcp --report` / `--module mcp-enforce --report`
- Result: `Result: SUCCESS` (both)

### Residual risks
- See Technical risks (§3) and Known gaps (§6) above — both accepted as
  deliberate scope boundaries, not oversights.

### Follow-ups

- [wont-do: out of scope] Redesign Execution Identity (ADR-007) to support a
  stdio-native binding mechanism. The current stdio/HTTP split is an
  intentional trust-boundary decision (see the MCP-conductor-harness ADR),
  not a gap this bugfix should close; `pose review attest`/`auto-attest`
  is the supported local path and is now properly cross-referenced.
- [wont-do: violates install invariant] Auto-repair `.pose/policy/
  {delivery,artifacts}.json` for instances already seeded with contaminated
  content. `pose install`/`update` deliberately never overwrite an existing
  policy file; `pose doctor`'s new detection is the correct-scoped mitigation.
