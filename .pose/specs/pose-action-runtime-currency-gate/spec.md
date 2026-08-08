---
slug: pose-action-runtime-currency-gate
status: done
created_at: 2026-08-07
completed_at: 2026-08-08
supersedes:
depends_on: pose-actions-node24-bump
priority: 2
components: release
delivers: governance:action-runtime-currency-gate
---

# Spec: A deprecated action runtime fails the build instead of annotating it

## 1. Intent

### Goal
Fail CI when a workflow references an action whose `action.yml`
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
- Detecting deprecations GitHub has not announced. The gate enforces a declared
  list; it does not predict.

**Scope changed during implementation.** The draft excluded third-party actions,
reasoning that they are SHA-pinned and reviewed on their own schedule. That was
reversed, and the first run showed why: `goreleaser/goreleaser-action` was still
on node20 — the only remaining deprecated runtime in the repository — precisely
because the Node 24 bump had covered first-party actions only. Excluding
third-party would have shipped a currency gate blind to the single thing it had
to find.

---

## 2. Requirements

### Functional
- R1: A deterministic offline check shall fail when any workflow references an
  action whose recorded runtime appears in the declared deprecated set.
- R2: Every non-local `uses:` reference in `.github/workflows/` shall have a
  recorded runtime pinned at the same ref; a reference with no record, or a
  record whose ref is stale, shall fail the same check — so neither adding an
  action nor bumping one can silently bypass the gate.
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
- `pose-mcp/internal/version/action_runtime_test.go` — a sibling of the pinning
  contract in the same package, so a pinning failure and a runtime failure stay
  distinguishable.
- `tests/release/action-runtime-verify.sh` — the online half.
- `.github/action-runtimes.json` — new: the recorded runtime per action
  reference plus the deprecated set.
- `.github/workflows/ci.yml` — the R3 step.

### Artifacts
- created: .pose/specs/pose-action-runtime-currency-gate/spec.md
- created: .github/action-runtimes.json
- created: pose-mcp/internal/version/action_runtime_test.go
- created: tests/release/action-runtime-verify.sh
- modified: .github/workflows/ci.yml
- modified: .github/workflows/release.yml

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
- [x] Confirm `runs.using` is present in every referenced action's manifest,
      including the composite ones, and decide their representation
- [x] Decide whether the check lives with the pinning contract or beside it

### Implementation
- [x] R1: fail on a recorded runtime in the deprecated set
- [x] R2: fail on a reference with no record
- [x] R4: manifest schema with owner + announced date; actionable message
- [x] R3: CI step resolving each `action.yml` at its pinned ref

### Validation
- [x] Prove R1 against a known-bad record
- [x] Prove R2 against a removed record and R3 against a stale ref
- [x] Prove the online half catches a deliberately wrong record
- [x] Run the mandatory checks

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
- Scope: offline manifest, deprecated set, unrecorded references and stale refs
- Expected: pass

#### Security / Contract
- Command: `bash tests/release/action-runtime-verify.sh`
- Scope: each recorded runtime against the action's own action.yml at its pinned ref
- Expected: pass

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64; runtimes resolved through the GitHub contents API at
  each action's pinned ref.
- Notes: building the manifest immediately found the defect the gate exists for
  — `goreleaser/goreleaser-action` was pinned at v6, which declares node20. Its
  v7.0.0 release is literally "node 24, update deps, rm yarn, ESM", so it was
  bumped to v7.2.3 and the repository now has no deprecated runtime. Fifteen
  actions are recorded: 11 node24, 3 composite, 1 docker.
- Each failure path was exercised against the real manifest rather than a
  fixture: recording node20 for goreleaser reports the deprecation and its
  announcement date; deleting the `actions/checkout` record reports it as
  unchecked; replacing `actions/setup-go`'s ref with zeros reports the record as
  describing a different version. The online half, given a record claiming
  node20 for `actions/checkout`, reports that the action declares node24 at its
  pinned ref.

### Results summary
- Successes: every failure path demonstrated; the only deprecated runtime in the
  repository found and removed
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] governance:action-runtime-currency-gate evidence:integration check:delivery-integration test:pose-mcp/internal/version/action_runtime_test.go — recording node20 for goreleaser produces a failure naming the action, the runtime and the 2025-09-19 announcement date
- R2 [satisfied] test:pose-mcp/internal/version/action_runtime_test.go — a removed record reports the action as unchecked and names the file to edit; a stale ref reports that the record describes a different version; a record for an unreferenced action is reported as stale bookkeeping
- R3 [satisfied] check:action-runtime-verify report:.github/workflows/ci.yml — the CI step resolves each action.yml at its pinned ref; run locally it confirms all 15 records and, given a deliberately wrong one, reports the disagreement
- R4 [satisfied] report:.github/action-runtimes.json — each deprecated entry carries runtime, owner, announced date and justification, and the check fails when any is missing

### Known gaps
- The deprecated set is a declared input. The gate cannot discover that a
  runtime became deprecated; it enforces what someone wrote down. That narrows
  the window from "nobody noticed" to "nobody updated the list" — an
  improvement, not a solution.
- The offline half trusts the record. Only the online step compares it to
  reality, and it needs network and a token, so a local `go test` run proves
  internal consistency rather than truth.
- A bump is a two-file edit by construction: the ref check fails until the
  record is refreshed. That is deliberate noise, but it is friction.

---

## 7. Final Report

### Delivered scope
A deprecated action runtime now fails the build. The offline check enforces the
declared deprecated set, the recorded ref and full coverage of referenced
actions; the CI step keeps the record honest against each action's own
action.yml.

### Files and modules changed
- .github/action-runtimes.json
- pose-mcp/internal/version/action_runtime_test.go
- tests/release/action-runtime-verify.sh
- .github/workflows/ci.yml, .github/workflows/release.yml

### Validation executed
- Command: the offline test with three injected failures; the online verifier
  against all 15 actions and against a deliberately wrong record
- Result: pass; every failure path named its cause

### Residual risks
- The gate is only as current as its deprecated list, which is human input.

### Follow-ups

- [open] If the recorded-runtime manifest proves to be pure overhead in practice — R3 never catching a drift R1 would have missed — collapse the two into the online check alone and delete the manifest. Decide on evidence after two or three bump cycles, not on principle. (owner:@pose-maintainers crit:low review:2027-02-05)
