---
slug: pose-adaptive-rule-delivery
status: in-progress  # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 3
components: pose-mcp
depends_on: pose-domain-rule-extension-migration, pose-stack-detection-consolidation
delivers: capability:adaptive-rule-delivery
---

# Spec: pose-adaptive-rule-delivery

> Single POSE spec template. Fill the relevant sections; remove the ones that
> don't apply. Keep the order: Intent → Requirements → Technical Plan →
> Tasks → Decisions → Validation → Final Report.
>
> **Lifecycle:** update `status` as you go (`draft` → `in-progress` → `done`).
> On completion, run the closeout flow (skill `pose-spec-closeout`): set
> `status: done`, fill `completed_at` and disposition every follow-up.

---

## 1. Intent

### Goal
Connect detected stack (`pose-stack-detection-consolidation`) to a concrete
rule-extension install decision (`pose-domain-rule-extension-migration`),
so a fresh install receives the domain rule matching its actual stack
instead of either a wrong default or nothing at all.

### Business value
`github.com/oseiaspereira88/pose#21` (adaptive rule delivery) and
`github.com/oseiaspereira88/pose#24` (extension-based delivery) both name
this outcome, but neither is buildable in isolation: adaptively installing
"the right rule for the stack" is not a well-defined operation until the
candidate rules are individually installable, versioned artifacts (which
`pose-domain-rule-extension-migration` produces) resolved against a real
detection signal (which `pose-stack-detection-consolidation` produces).
This spec is deliberately the join of the two, not a restatement of either.

Scope is intentionally bounded to whatever extensions exist by the time
this spec starts — initially `pose-rule-backend-go`,
`pose-rule-frontend-react`, and the pre-existing `pose-rule-kubernetes`.
It does not require or assume a general catalog: a small, in-repo,
curated stack-to-extension mapping is sufficient and matches what actually
exists today.

### Constraints
- Cannot start implementation before both `pose-domain-rule-extension-migration`
  and `pose-stack-detection-consolidation` reach `status: done` — the
  roadmap's `adaptive-delivery` milestone declares this dependency
  explicitly (`after: rule-extensionization, stack-detection`).
- Must not fail or block install/doctor when a detected stack has no
  matching extension yet (e.g. Rust, Python) — recommending zero domain
  rules and saying nothing is the correct behavior until those extensions
  are authored, not an error.
- `pose extension install <package-dir>` takes a local directory only —
  confirmed by reading `extension.go`, no URL fetch, no catalog/registry.
  `extensions/` is also not in `distpolicy.IncludedTopLevel`, so it is not
  embedded in the scaffold dist shipped to consumer instances either. See
  Decision 1: this makes automatic install unreachable for an external
  consumer today, and reshapes this spec's actual deliverable.

### Non-goals
- Authoring new stack extensions beyond what
  `pose-domain-rule-extension-migration` already produces — adding Rust/
  Python/mobile/etc. rule extensions is future content work, not part of
  wiring the resolution mechanism.
- Building catalog/registry/discovery infrastructure — the mapping this
  spec needs is small and curated, not discovered.

---

## 2. Requirements

> Definition of Ready (entry gate): before `status: in-progress`, functional
> requirements must have **acceptance criteria with stable IDs** (`- R<N>: ...`).
> Published IDs are never renumbered; a removed criterion is marked as
> withdrawn. Verify with `pose lint-spec <slug> --ready-check`.
>
> Optional EARS form: `- R1: When <trigger>, the <system> shall <behavior>.`
> Verify an opted-in spec with `pose lint-spec <slug> --ears`.

### Functional
- R1: for each module `pose-stack-detection-consolidation` records in
  `module-metadata.json`, `pose doctor` shall resolve a curated
  stack-to-extension mapping and, when a match exists and that rule is not
  already installed locally, surface an advisory recommendation naming the
  exact extension ID and the `pose extension install <path>` command to
  run — see Decision 1 for why this is advisory rather than automatic
  install.
- R2: when a detected stack has no matching extension, the advisory shall
  say nothing for that module — silent, not an error and not a false
  recommendation.
- R3: `node` shall resolve to `pose-rule-frontend-react` only when the
  module's `package.json` actually lists `react` as a dependency or
  dev-dependency — a Node.js backend with no React must never be
  recommended the React rule. `go` shall always resolve to
  `pose-rule-backend-go`.

### Compatibility
- Additive for fresh installs. Does not retroactively install extensions
  into already-existing instances — an instance that predates this spec
  keeps whatever `.pose/rules/` content it already has until it next runs
  through an install/update path this spec touches.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/rule_extension_resolver.go` (new — curated
  stack-to-extension mapping, `resolveRuleExtension`)
- `pose-mcp/internal/cli/doctor.go` (new advisory check)

### Artifacts
- created: pose-mcp/internal/cli/rule_extension_resolver.go
- created: pose-mcp/internal/cli/rule_extension_resolver_test.go
- modified: pose-mcp/internal/cli/doctor.go
- created: .pose/changelogs/unreleased/pose-adaptive-rule-delivery.md

### Delivery targets
- capability:adaptive-rule-delivery module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: pure resolution function plus one advisory doctor check, no
  install-time behavior change and no new install-time failure mode.

---

## 4. Tasks

### Planning
- [x] Confirm this spec is a join of two prerequisites, not independently
      buildable (issue #21/#24 investigation, this repo, 2026-08-15)
- [x] Both prerequisites reached `status: done` (2026-08-15)
- [x] Discovered `pose extension install` has no fetch/catalog capability
      and `extensions/` is not embedded in the scaffold dist — reshapes R1
      from "auto-install" to "advisory recommendation," recorded as
      Decision 1

### Implementation
- [x] `resolveRuleExtension(root, modulePath, stack string) (string, bool)`:
      `go` always resolves to `pose-rule-backend-go`; `node` resolves to
      `pose-rule-frontend-react` only when `<modulePath>/package.json`
      actually lists `react` as a dependency/dev-dependency (R3); every
      other stack resolves to nothing (R2)
- [x] New `pose doctor` check `rules.stack-extension-available`: for every
      module in `module-metadata.json`, resolves the extension and, when
      matched and not already present at `.pose/rules/<file>.md`, emits an
      advisory naming the extension ID and the install command (R1)

### Validation
- [x] `go -C pose-mcp test ./...`, `go vet ./...`, `gofmt -l .`: all clean
- [x] Unit tests for `resolveRuleExtension`: Go always matches; Node with
      `react` dependency matches; Node without it does not; Rust/Python/
      unknown stacks never match
- [x] `pose doctor` regression tests: advisory fires for an unmatched-but-
      resolvable module, stays silent for an already-installed rule and
      for an unmapped stack

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: R1 originally called for "offer to install... auto-install a
  baseline or prompt... via the existing `pose extension install`
  mechanism." Reading `extension.go` while starting implementation:
  `pose extension install <package-dir>` accepts only a local directory
  path — no URL fetch, no catalog/registry, confirmed in the earlier
  `pose-domain-rule-extension-migration` investigation. Separately,
  `distpolicy.IncludedTopLevel` (the scaffold-embedding allowlist) does not
  include `extensions/` at all, so `extensions/pose-rule-backend-go/` and
  `extensions/pose-rule-frontend-react/` are not shipped inside the `pose`
  binary to any consumer instance — they only exist in this repository's
  own working tree. An external consumer's `pose install`/`pose doctor`
  therefore has no path it could pass to `pose extension install` even if
  it wanted to auto-install.
- Options considered: (a) build a minimal fetch mechanism (e.g. download
  from a GitHub release URL) as part of this spec, so auto-install becomes
  literally reachable; (b) scope this spec down to resolution +
  recommendation only, matching the level of automation that actually
  exists today (none) for `pose-rule-kubernetes` too — `AGENTS.md` already
  tells an operator to `pose extension install` a kubernetes rule with no
  built-in way to obtain it either.
- Decision: (b). Implemented `resolveRuleExtension` (pure, testable
  mapping) plus a `pose doctor` advisory that names the matching extension
  and the install command — informational, consistent with every other
  `pose doctor` check's contract (never blocking, never mutating).
- Rationale: (a) is a materially larger feature (fetch, verify, cache,
  probably a real catalog/registry) that both
  `pose-domain-rule-extension-migration` and this spec's own Non-goals
  already named as explicitly out of scope — building it now, discovered
  mid-implementation rather than planned, would be exactly the kind of
  scope creep this session's specs have consistently pushed back on
  elsewhere (see `pose-monorepo-validation-advisory`'s Decision 1,
  `pose-domain-rule-extension-migration`'s Non-goals). An advisory that
  correctly names what to install is real, shippable value; a half-built
  fetch mechanism would not be.
- Consequences: R1/R3 (spec wording) were revised in place before this
  spec's first `status: in-progress` closeout — not amended after the
  fact — to describe the advisory behavior actually delivered. Building
  real fetch/catalog infrastructure so the recommendation becomes a single
  runnable command remains explicitly open (Follow-ups).

---

## 6. Validation

### Strategy
Deterministic unit tests for the pure resolution function (the part with
real branching logic: the react-dependency check) plus `pose doctor`
regression tests for the advisory's three states (recommend / already
installed / no match).

### Requirement trace
- R1 [satisfied] `pose doctor`'s `rules.stack-extension-available` check;
  see Decision 1 for the revised (advisory, not auto-install) scope.
- R2 [satisfied] unit tests confirm Rust/Python/unmapped stacks resolve to
  no recommendation.
- R3 [satisfied] unit tests confirm `go` always matches and `node` matches
  only with an actual `react` dependency.

### Known gaps
- No fetch/catalog mechanism exists to turn the advisory into a single
  runnable command for an external consumer — Decision 1, tracked as a
  follow-up.

---

## 7. Final Report

### Delivered scope
Joins stack detection to rule-extension awareness: `resolveRuleExtension`
maps a detected module to the correct extension (with a real
react-dependency check for Node, not a blind stack-name match), and `pose
doctor` recommends installing it when appropriate. Discovered mid-
implementation that literal auto-install is unreachable today (no
fetch/catalog mechanism, `extensions/` not embedded in the scaffold dist)
— scoped down to advisory recommendation, matching the level of automation
`pose-rule-kubernetes` already has (Decision 1).

### Files and modules changed
- `pose-mcp/internal/cli/rule_extension_resolver.go`: new,
  `resolveRuleExtension`.
- `pose-mcp/internal/cli/rule_extension_resolver_test.go`: new.
- `pose-mcp/internal/cli/doctor.go`: `rules.stack-extension-available`
  check.

### Validation executed
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l .`: clean.

### Residual risks
- None identified for what shipped. The advisory only recommends by
  name — an operator still needs to source the extension package
  themselves, same as `pose-rule-kubernetes` today.

### Follow-ups

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [open] non-Go/React/Kubernetes stack extensions (Rust, Python, mobile,
  Cloudflare Workers, etc.) — this spec wires resolution for whatever
  extensions exist; authoring new ones is separate content work with no
  spec of its own yet.
- [open] a fetch/catalog mechanism so `pose extension install` can resolve
  an extension ID to a real package without the operator sourcing it
  manually — Decision 1's deferred scope, needed before the doctor
  advisory can become a single runnable command. 
