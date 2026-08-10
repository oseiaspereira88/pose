---
slug: pose-scaffold-exclusion-policy
status: done
created_at: 2026-08-10
completed_at: 2026-08-10
supersedes:
depends_on: pose-composition-contract
priority: 1
components: pose-mcp
delivers: governance:scaffold-exclusion-policy
---

# Spec: The scaffold's exclusion list stops existing twice

## 1. Intent

### Goal
Stop `composition-contract.json` from being distributed to every POSE instance,
and make the exclusion list a single source the generator and the drift guard
both read.

### Business value
`pose-composition-contract` published a file describing how to compose *this
product's* images — ports, build contexts, service environment. The scaffold
generator copies everything not explicitly excluded, so it went into the
embedded `dist/` and would have reached every instance on the next
`pose upgrade`: a project that installed POSE would receive a declaration about
services it does not run.

That is precisely the category `compatibility.json` was already excluded for,
and the category a standing follow-up names for `frontend-react.md` and
`backend-go.md`, which "ship to every instance for the same bad reason
`kubernetes.md` did". The generator's default is inclusion, so every new
product-level file at the repository root is one omission away from being
distribution.

Fixing it exposed a second defect. The exclusion list existed **twice** — once
in `gen/main.go` and once in `scaffold_test.go`, the latter annotated
`// Mirrors gen/main.go`, which is an admission rather than a mechanism. Adding
the exclusion to the generator alone made it emit a tree the guard immediately
rejected, so the two-line fix failed on its first run. The comment had been
right about the risk and powerless against it.

This is the same shape as the spec that caused it: knowledge written down in two
places with nothing reconciling the copies. Duplicating one more entry here,
having just shipped a contract against exactly that, would have been
incoherent.

### Constraints
- The shared package cannot live in `scaffold`, which embeds `dist/` through
  `go:embed`. The generator *creates* that directory, so it must not depend on
  a package requiring it to already exist.

### Non-goals
- Auditing the rest of the embedded root. `README.md`, `CONTRIBUTING.md`,
  `SECURITY.md`, `LICENSE` and `scripts/` are also product-level and were
  already embedded before this change; whether they belong is a separate
  question with its own answer, recorded as a follow-up rather than resolved
  under a bugfix.

---

## 2. Requirements

### Functional
- R1: `composition-contract.json` shall not appear in the embedded scaffold,
  while remaining published at the repository root for consumers.
- R2: The exclusion list shall exist once, consumed by both the generator and
  the drift guard.
- R3: Adding an exclusion in that one place shall take effect on both sides
  without editing either.

### Non-functional
- No behaviour change for any other embedded path.

### Security
- None; the change narrows what is distributed.

### Compatibility
- Instances lose nothing: the file had never been released, so no upgrade
  removes anything a user already had.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/scaffold/distpolicy` — new, the single list.
- `pose-mcp/internal/scaffold/gen/main.go` — consumes it.
- `pose-mcp/internal/scaffold/scaffold_test.go` — consumes it, dropping the
  copy that said it mirrored the generator.

### Artifacts
- created: .pose/specs/pose-scaffold-exclusion-policy/spec.md
- created: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/gen/main.go
- modified: pose-mcp/internal/scaffold/scaffold_test.go

### Delivery targets
- governance:scaffold-exclusion-policy module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None public. `distpolicy` is internal.

### Technical risks
- The generator still includes by default. A future product-level file at the
  repository root is distributed unless someone remembers to exclude it — the
  list is now single, but it is still a denylist. An allowlist would invert the
  failure mode, at the cost of every new scaffold file needing registration.

---

## 4. Tasks

### Planning
- [x] Confirm the file was embedded and would ship on the next upgrade
- [x] Establish why the exclusion could not simply be added in one place

### Implementation
- [x] R1: exclude the published contract from the embedded scaffold
- [x] R2: one list, consumed by generator and guard
- [x] R3: an exclusion added once applies to both

### Validation
- [x] Confirm the file left the embed and stayed at the root
- [x] Prove a new exclusion takes effect on both sides at once
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
Two things must hold and only one is obvious. The file must leave the embedded
tree while remaining published — a fix that removed it from both would satisfy
the symptom and destroy the deliverable. And the deduplication must be real:
adding an exclusion in the shared package has to change the generator's output
*and* satisfy the guard, with neither file edited.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/scaffold/... -count=1`
- Scope: the embed drift guard against the regenerated tree
- Expected: pass

### Execution log
- Date: 2026-08-10
- Environment: linux/amd64.
- Notes: after excluding it, `pose-mcp/internal/scaffold/dist/composition-contract.json`
  is gone and `composition-contract.json` remains at the repository root (1806
  bytes), with its own contract check still passing.
- The deduplication was proven by adding `README.md` to `distpolicy` and
  touching nothing else: the generator stopped emitting it and the drift guard
  passed on that tree in the same run. Reverting restored both. Before this
  change the same experiment required editing two files, and doing one broke
  the build — which is how the defect was found.

### Results summary
- Successes: the contract is no longer distributed to instances; the exclusion
  list has one home and both consumers follow it
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:scaffold-exclusion-policy evidence:integration check:delivery-integration test:pose-mcp/internal/scaffold/scaffold_test.go — the file is absent from the embedded tree and present at the repository root, where the composition contract check still reads it
- R2 [satisfied] test:pose-mcp/internal/scaffold/scaffold_test.go — the shared `distpolicy` package is called by both the generator and the guard; the guard's own copy, and its "Mirrors gen/main.go" comment, are gone. (`pose-scaffold-allowlist` later replaced the exclusion helpers with `IsIncluded`; the single-source property this requirement asserts is unchanged.)
- R3 [satisfied] check:embedded-dist-drift — adding `README.md` to the shared list changed the generator's output and kept the guard green without editing either file

### Known gaps
- The generator remains a denylist: inclusion is the default, so the next
  product-level file at the repository root is distributed unless someone
  remembers.
- Other product-level files already embedded (`README.md`, `CONTRIBUTING.md`,
  `SECURITY.md`, `LICENSE`, `scripts/`) were not audited here.

---

## 7. Final Report

### Delivered scope
The published composition contract stays out of the embedded scaffold, and the
exclusion list is one shared package instead of two copies annotated as mirrors.

### Files and modules changed
- pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- pose-mcp/internal/scaffold/gen/main.go
- pose-mcp/internal/scaffold/scaffold_test.go

### Validation executed
- Command: the scaffold suite, plus an injected exclusion applied to both sides
- Result: pass; the injected exclusion took effect on generator and guard at once

### Residual risks
- Default-include means the next product-level root file repeats this unless
  excluded.

### Follow-ups

- [covered: pose-scaffold-allowlist] The scaffold generator is a denylist: everything at the repository root is embedded unless named. That default is why a published product contract nearly shipped to every instance, and why `frontend-react.md`/`backend-go.md` still do. Consider inverting it to an allowlist, where a new scaffold file must be registered to be distributed — the failure mode becomes "a new scaffold file is missing" rather than "a product file was silently distributed". (owner:@pose-maintainers crit:medium review:2026-10-10)
- [covered: pose-scaffold-allowlist] Audit the product-level files already embedded — `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE` and `scripts/` — and decide which an instance actually needs. `scripts/` in particular now carries release and maintenance tooling for this product (`release.sh`, `refresh-action-runtimes.sh`) that no instance runs. (owner:@pose-maintainers crit:low review:2026-11-13)
