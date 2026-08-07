---
slug: pose-action-runtime-currency-gate
status: draft
created_at: 2026-08-07
completed_at:
supersedes:
depends_on: pose-actions-node24-bump
priority: 2
components: release
delivers: governance:action-runtime-currency-gate
---

# Spec: A deprecated action runtime fails the build instead of annotating it

## 1. Intent

### Goal
Fail CI when a workflow references a first-party action whose `action.yml`
declares a runtime GitHub has deprecated, so the next deprecation is caught by a
gate rather than by someone reading an annotation on a green run.

### Business value
The Node.js 20 deprecation was not detected by any check. It was detected
because a human read the annotation block under a passing run, weeks after the
runner had started silently substituting Node 24. Until then every gate was
green and every workflow was, in fact, one GitHub-controlled withdrawal away
from failing at once — CI, release, verification and the security scanners
together.

That is the shape of the risk worth spending a gate on: not a failure, but a
correct-looking success that depends on someone else's temporary kindness. The
signal existed the whole time and was machine-readable the whole time; nothing
was watching it.

The cost of missing it again is asymmetric. A stale pin costs one bump, done
calmly. The same pin discovered when the substitution is withdrawn costs a
release blocked at the worst moment, which is exactly the failure mode that
`pose-shellcheck-ci-gate` was written about in a different guise.

### Constraints
- The offline check must not need network access. `go test ./...` is run
  locally and in the governance job; a check that reaches the GitHub API on
  every run would make the suite fail on a plane and rate-limit in CI.
- The pinning contract is unchanged: first-party by tag, third-party by full
  commit SHA. This gate reads what is pinned; it does not repin anything.
- Deprecation dates are a human input. Nothing in `action.yml` says a runtime
  is deprecated — only which runtime is used — so the list of deprecated
  runtimes is declared, owned and dated, like `.github/security-exceptions.json`.

### Non-goals
- Auto-bumping actions. The gate reports; a human decides the target major,
  because a major bump can carry behaviour changes (as `setup-go@v6`'s
  toolchain handling and `checkout@v7`'s fork-PR block both did).
- Covering third-party actions' runtimes. They are SHA-pinned and reviewed on
  the pinning exception's schedule; widening scope here dilutes the signal.
- Detecting deprecations GitHub has not announced. The gate enforces a declared
  list; it does not predict.

---

## 2. Requirements

### Functional
- R1: A deterministic offline check shall fail when any workflow references a
  first-party action whose recorded runtime appears in the declared deprecated
  set.
- R2: Every first-party `uses:` reference in `.github/workflows/` shall have a
  recorded runtime; a reference with no record shall fail the same check, so
  adding an action cannot silently bypass the gate.
- R3: A CI step shall resolve each referenced action's `action.yml` at its
  pinned ref and fail when the real `runs.using` differs from the recorded one,
  so the record cannot drift from reality unnoticed.
- R4: The deprecated set shall carry, per entry, an owner and the date the
  runtime was announced as deprecated, and the failure message shall name the
  action, its recorded runtime and the workflow it appears in.

### Non-functional
- The offline check adds no measurable time to `go test ./...`: it parses eight
  YAML files and one JSON file.
- R3's step is one network call per distinct action reference — under twenty,
  deduplicated — and runs on push, not per job.

### Security
- The manifest is repository-controlled data read by a test; it grants no
  authority and executes nothing.
- R3's step reads public `action.yml` contents with the workflow's own token
  and must not be given write scopes.

### Compatibility
- No product change. `TestWorkflowSecurityContract` keeps its current
  responsibilities; this is a sibling check, not an extension of it, so a
  pinning failure and a runtime failure stay distinguishable.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/version/workflow_security_test.go` — or a sibling file in
  the same package, reusing `usesRe` and `firstPartyOwners`.
- `.github/action-runtimes.json` — new: the recorded runtime per action
  reference plus the deprecated set.
- `.github/workflows/ci.yml` — the R3 step.

### Artifacts
- created: .pose/specs/pose-action-runtime-currency-gate/spec.md
- created: .github/action-runtimes.json
- modified: pose-mcp/internal/version/workflow_security_test.go
- modified: .github/workflows/ci.yml

### Delivery targets
- governance:action-runtime-currency-gate module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None. `.github/action-runtimes.json` is a new repository-local file with no
  consumer outside the check.

### Data/storage changes
- One JSON manifest, shaped after `.github/security-exceptions.json`:
  a `runtimes` map from `owner/repo@ref` to the observed `runs.using`, and a
  `deprecated` list of runtime identifiers each with `owner`, `announced` and a
  justification. Composite actions record `composite` and are exempt from the
  deprecated check by construction.

### Technical risks
- **The manifest is a second source of truth.** Its whole failure mode is
  drifting from the real `action.yml`, which is precisely why R3 exists; without
  R3 this gate is theatre. R3 is the requirement to defend under time pressure,
  not R1.
- **The deprecated list is only as fresh as its last edit.** The gate cannot
  discover that a runtime became deprecated; it enforces what was declared. This
  narrows the window from "nobody noticed" to "nobody updated the list", which
  is an improvement, not a solution. Worth stating plainly in the spec's known
  gaps rather than implying the problem is closed.
- **Ref-keyed records go stale on every bump.** Keying by `owner/repo@ref` means
  a bump invalidates the entry and R2 fails until the record is updated — noisy
  by design, and the noise is the point, but it makes the bump a two-file edit.

---

## 4. Tasks

### Planning
- [ ] Confirm `runs.using` is present in every referenced action's manifest,
      including the composite ones, and decide their representation
- [ ] Decide whether the check lives with the pinning contract or beside it

### Implementation
- [ ] R1: fail on a recorded runtime in the deprecated set
- [ ] R2: fail on a first-party reference with no record
- [ ] R4: manifest schema with owner + announced date; actionable message
- [ ] R3: CI step resolving each `action.yml` at its pinned ref

### Validation
- [ ] Prove R1 against a known-bad fixture: a workflow pinning a node20 major
- [ ] Prove R3 catches a deliberately wrong record
- [ ] Run the mandatory checks

---

## 6. Validation

### Strategy
Prove the gate fails before trusting that it passes. A check of this kind is
worth exactly as much as its demonstrated failure path — the lesson
`pose-release-signing-rejection` recorded for the signing gate applies here
unchanged, and this spec should not close with its negative path asserted.

Two negative fixtures, not one: a workflow referencing a node20 major (R1), and
a manifest whose recorded runtime contradicts the real `action.yml` (R3). The
second is the one that matters, because it is the failure mode the manifest
itself introduces.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/version/... -run ActionRuntime -count=1`
- Scope: offline manifest and deprecated-set enforcement, with negative fixtures
- Expected: pass, including both rejection cases

#### Security / Contract
- Command: `go -C pose-mcp test ./internal/version/... -run WorkflowSecurity -count=1`
- Scope: the existing pinning contract, unchanged by this work
- Expected: pass

### Execution log
<!-- Filled at implementation; nothing has been executed. -->

### Results summary
<!-- Filled at implementation. -->

### Requirement trace
<!-- Filled at implementation; R3's evidence is a real CI run, not a local command. -->

### Known gaps
<!-- Filled at implementation. Expected to record that the deprecated set is a
     declared input and the gate cannot discover a new deprecation on its own. -->

---

## 7. Final Report

### Delivered scope
<!-- Filled at closeout. -->

### Files and modules changed
<!-- Filled at closeout. -->

### Validation executed
<!-- Filled at closeout. -->

### Residual risks
<!-- Filled at closeout. -->

### Follow-ups

- [open] If the recorded-runtime manifest proves to be pure overhead in practice — R3 never catching a drift R1 would have missed — collapse the two into the online check alone and delete the manifest. Decide on evidence after two or three bump cycles, not on principle. (owner:@pose-maintainers crit:low review:2027-02-05)
