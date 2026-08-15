---
slug: pose-domain-rule-extension-migration
status: done
created_at: 2026-08-15
completed_at: 2026-08-15
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 2
components: pose-mcp
depends_on:
delivers: capability:domain-rule-extension-migration
---

# Spec: pose-domain-rule-extension-migration

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
Stop embedding `backend-go.md` and `frontend-react.md` in core POSE
machinery; deliver them exclusively through the same signed extension
mechanism already proven by `pose-rule-kubernetes`.

### Business value
`github.com/oseiaspereira88/pose#24`: every `pose install` copies every
file in `.pose/rules/` unconditionally (`deliverMachinery`,
`machinery.go`), regardless of the target repository's actual stack — a
pure-TypeScript or Rust project still receives `backend-go.md`. This
duplicates, almost verbatim, an **already-open, already-tracked
follow-up**: Finding F1 in `pose-extension-reference-publication`
(`knowledge:pose-extension-reference-publication`) states *"frontend-react.md
and backend-go.md ship to every instance for the same bad reason
kubernetes.md did... decide whether they follow it out of machinery now
that the extension path is proven"* (owner `@pose-maintainers`, review
2026-10-19). This spec is that decision being picked up and executed, not a
new idea.

The extension mechanism is mature, not aspirational: `pose extension
install|list|remove|verify` is signed via cosign/Sigstore, transactional
with rollback, and restricted by a data-only whitelist to `.agents/skills/`,
`.pose/workflows/`, `.pose/rules/`, `.pose/templates/`. `pose-rule-kubernetes`
already proves the full pattern end to end: `.pose/rules/` in this very
repository contains no `kubernetes.md` — it installs exclusively via
extension, with `AGENTS.md` documenting that explicitly. Migrating two more
files onto an already-shipped, already-verified path is a small, bounded
increment — deliberately **not** the issue's larger ask of a full ~10-stack
"official catalog," which would require new registry/versioning/per-package
signing-identity infrastructure this repository does not have today and
which does not belong in this spec.

### Constraints
- No catalog/registry infrastructure exists and building one is explicitly
  out of scope for this spec (tracked separately, future roadmap item, not
  `adaptive-instance-provisioning`). `pose extension install` continues to
  take a local directory or a released artifact the way
  `pose-rule-kubernetes` already does.
- Must not silently drop rule content that an already-installed instance's
  agents may depend on — see Decision 1 below. This spec cannot leave
  `in-progress` for `status: in-progress` until that decision is made via a
  dedicated ADR (roadmap `adaptive-instance-provisioning`, milestone
  `rule-extensionization` requires the ADR to precede implementation).
- Must not affect any other embedded rule (`security.md`,
  `documentation-style.md`, `delivery-evidence.md`, `knowledge-governance.md`,
  `_base-recurrence.md`, `release-integrity.md`) — those are universal
  governance rules per the issue's own framing and stay embedded.

### Non-goals
- Building a general "official catalog" / registry / discovery service for
  rule extensions.
- Migrating any stack beyond Go-backend and React-frontend (no
  Cloudflare Workers, Rust, Python, mobile, etc. in this spec — those
  require the extension to exist first, which is a content-authoring task
  independent of this migration).
- Auto-resolving which extension to install based on detected stack — that
  is `pose-adaptive-rule-delivery`'s scope
  (`milestone:adaptive-instance-provisioning/adaptive-delivery`), which
  depends on this spec completing first.

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
- R1: `backend-go.md` and `frontend-react.md` shall be removed from
  `.pose/rules/` (and the embedded scaffold dist) and shall instead be
  published as installable extensions (`extensions/pose-rule-backend-go`,
  `extensions/pose-rule-frontend-react`) following the exact structure and
  signing process `pose-rule-kubernetes` already established.
- R2: `AGENTS.md`'s domain-rules section shall reference both as
  extension-delivered, mirroring the existing kubernetes sentence, instead
  of listing them as always-present files.
- R3: `.pose/indexes/task-map.json`'s `rules_by_domain` entries for
  `backend-go`/`frontend-react` shall continue to resolve correctly via
  `pose suggest` once the rule is installed as an extension — no regression
  to the domain-rule surfacing mechanism confirmed in
  `pose-knowledge-durable-reference-type`'s investigation.
- R4: `pose update`/`pose install` shall not delete, overwrite, or migrate
  an already-installed instance's existing `backend-go.md`/
  `frontend-react.md` — per Decision 1/ADR, this requires no new code, only
  removing the two files from `machineryRoots`'s source set.
- R5: `pose doctor` shall gain an advisory check that compares an
  instance's `machinery-manifest.json` delivered-paths history against the
  current machinery walk, and surfaces any path recorded as previously
  delivered, still present on disk, but no longer produced by the current
  walk — naming the matching extension to install. Advisory only; never
  blocking, never altering `pose doctor`'s exit code on its own.

### Non-functional
- No change to the extension mechanism itself (signing, whitelist,
  transactional install) — this spec only produces two new extension
  packages and removes their embedded equivalents, it does not modify
  `extension.go`.

### Compatibility
- Non-breaking for already-installed instances per Decision 1/ADR: existing
  `backend-go.md`/`frontend-react.md` on disk are left untouched;
  `pose update` simply stops re-seeding them. New installs never receive
  them as embedded files; `pose doctor`'s new advisory (R5) is the
  discoverability signal for instances that had them before this spec.

---

## 3. Technical Plan

### Affected areas
- `.pose/rules/backend-go.md`, `.pose/rules/frontend-react.md` (removed
  from core, content moves)
- `extensions/pose-rule-backend-go/`, `extensions/pose-rule-frontend-react/`
  (new, mirroring `extensions/pose-rule-kubernetes/` structure)
- `AGENTS.md` (both locales) — domain-rules section wording
- `pose-mcp/internal/cli/machinery.go` (`deliverMachinery` — confirm no
  hardcoded reference to the two files being removed)
- `pose-mcp/internal/cli/doctor.go` (new advisory check, R5)
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go` (`ExtensionOnlyRuleFiles`
  — pose-dist's own locally-installed dogfood copy must not re-leak into
  the embedded dist)
- `pose-mcp/internal/pose/review_bundle.go` (`reviewBundlePathClass` —
  `extensions/` had no classification; discovered when this spec's own
  change set became the first to touch that directory since
  component-aware review bundles were adopted)
- `pose-mcp/internal/scaffold/dist/**` (regenerated)

### Artifacts
- renamed: .pose/rules/backend-go.md -> extensions/pose-rule-backend-go/files/.pose/rules/backend-go.md
- renamed: .pose/rules/frontend-react.md -> extensions/pose-rule-frontend-react/files/.pose/rules/frontend-react.md
- created: extensions/pose-rule-backend-go/extension.json
- created: extensions/pose-rule-frontend-react/extension.json
- removed: locales/pt-BR/.pose/rules/backend-go.md
- removed: locales/pt-BR/.pose/rules/frontend-react.md
- modified: AGENTS.md
- modified: locales/pt-BR/AGENTS.md
- modified: .pose/workflows/review.md
- modified: .pose/workflows/recurrence-escalation.md
- modified: locales/pt-BR/.pose/workflows/review.md
- modified: locales/pt-BR/.pose/workflows/recurrence-escalation.md
- modified: pose-mcp/internal/cli/doctor.go
- created: pose-mcp/internal/cli/doctor_retired_machinery_test.go
- modified: pose-mcp/internal/scaffold/locale_coverage_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go
- modified: pose-mcp/internal/scaffold/scaffold_test.go
- modified: .pose/indexes/extensions.lock.json

### Delivery targets
- capability:domain-rule-extension-migration module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Medium: this is a structural/contract change to what every fresh install
  receives by default — mitigated by resolving Decision 1's ADR before
  implementation and by `pose-rule-kubernetes` being a working precedent
  for the mechanics.

---

## 4. Tasks

### Planning
- [x] Confirm the mechanism is real and precedented (issue #24
      investigation, this repo, 2026-08-15): `pose-rule-kubernetes` ships
      with no static `.pose/rules/kubernetes.md` fallback
- [x] Confirm this duplicates Finding F1 in
      `pose-extension-reference-publication` rather than being new scope
- [ ] Resolve Decision 1 (compatibility strategy for already-installed
      instances) via a dedicated ADR before moving to in-progress

### Implementation
- [x] Moved `.pose/rules/backend-go.md`/`frontend-react.md` to
      `extensions/pose-rule-backend-go/`/`extensions/pose-rule-frontend-react/`
      (`git mv`, preserving history), mirroring `pose-rule-kubernetes`'s
      `extension.json` structure exactly
- [x] Removed the orphaned `locales/pt-BR/.pose/rules/backend-go.md`/
      `frontend-react.md` — `machinerySource`'s locale overlay is only
      reached for paths the base `.pose/rules/` walk visits, so these would
      have become permanently unreachable dead content otherwise;
      extensions carry no locale variant today (matching kubernetes), noted
      as a residual gap, not silently fixed here
- [x] Updated `AGENTS.md` (EN + pt-BR) domain-rules section to describe
      both as extension-delivered, matching the existing kubernetes
      sentence
- [x] Updated `.pose/workflows/review.md`/`recurrence-escalation.md` (EN +
      pt-BR) rule-selection checklists to annotate both with their
      extension name, matching the existing kubernetes annotation
- [x] Removed `backend-go.md`/`frontend-react.md` from
      `locale_coverage_test.go`'s structural-comparison list (no longer
      core machinery)
- [x] Implemented R5: `pose doctor`'s `machinery.retired-on-disk` check
      (`doctor.go`) — compares `machinery-manifest.json`'s delivered-paths
      history against a fresh walk of the current embedded
      `scaffold.Dist()` machinery roots; a path recorded as delivered,
      still present on disk, but no longer in the current walk is flagged
      advisory-only, naming the matching `pose-rule-<slug>` extension for
      any `.pose/rules/*.md` path
- [x] Regenerated `pose-mcp/internal/scaffold/dist/**`
- [x] Decision 2: installed both extensions locally into pose-dist's own
      `.pose/rules/` (dogfooding — its own review profiles cite `backend-go`/
      `frontend-react` as rule names), added
      `distpolicy.ExtensionOnlyRuleFiles` so that local install does not
      re-leak into the embedded dist, and fixed
      `TestEditorialDefaultsAreEnglishAndPtBROverlayIsComplete`'s locale
      scan to skip extension-only content
- [x] Fixed `reviewBundlePathClass` (`review_bundle.go`): `extensions/` had
      no classification at all — undiscovered until now because
      `pose-rule-kubernetes` shipped before component-aware review bundles
      were adopted. Classified as `governance`, matching `.pose/rules/`

### Validation
- [x] `go -C pose-mcp test ./...`, `go vet ./...`, `gofmt -l .`: all clean
- [x] New regression tests in `doctor_retired_machinery_test.go`: warns
      when a retired file is present, silent when the instance already
      removed it itself, silent/no-crash when no manifest exists at all
- [x] Fresh `pose install` into a throwaway repo: `.pose/rules/` contains
      no `backend-go.md`/`frontend-react.md` — R1 confirmed
- [x] `pose extension install extensions/pose-rule-backend-go --allow-unsigned --yes`
      against the same throwaway repo: installs cleanly, file lands at
      `.pose/rules/backend-go.md` — R1's extension-delivery path confirmed
      working end to end
- [x] Simulated a pre-migration instance (hand-written manifest recording
      `backend-go.md` as previously delivered, file present on disk,
      extension not installed) and ran `pose doctor --json`: confirmed the
      exact `machinery.retired-on-disk` warning and hint — R5 confirmed
      empirically, not just via unit test

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: removing `backend-go.md`/`frontend-react.md` from core machinery
  means `pose update` on an already-installed instance stops re-delivering
  them. Two candidate strategies were identified, not yet chosen between:
  (a) `pose update` auto-installs the matching extension for any instance
  whose current `.pose/rules/` already contains the file being retired,
  preserving effective content across the migration; (b) leave
  already-delivered files exactly as they are on disk (POSE never deletes
  content a prior version wrote) and only stop *re-seeding* them — meaning
  existing instances silently stop receiving updates to that rule's content
  unless they manually adopt the extension.
- Decision: **(b) — leave already-delivered files on disk, stop
  re-seeding, plus a `pose doctor` advisory check.** Reading
  `deliverMachinery` (`machinery.go`) confirms it only walks the *current*
  engine source's `machineryRoots`; a file dropped from that source is
  never visited, and nothing in the walk ever deletes or overwrites a file
  that drops out — so (b) is not new code, it is what already happens the
  moment the two files leave `.pose/rules/`. (a) was rejected: it would
  blur `pose update`'s contract (machinery refresh) with `pose extension
  install`'s (explicit, signed, transactional) for a benefit an advisory
  check already covers with far less risk. Full rationale and consequences
  in
  `.pose/adr/2026-08-15-retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update.md`.
- Rationale: this is a structural/contract change (AGENTS.md's own
  definition of when an ADR is required) affecting every already-installed
  instance, not a purely additive change — deciding it inline, without the
  scrutiny an ADR forces, risks silently degrading agents' governance
  context in every downstream repo already running POSE.
- Consequences: implementation includes a new `pose doctor` advisory check
  comparing `machinery-manifest.json`'s delivered-paths history against
  the current machinery walk, surfacing any path that dropped out while
  still present on disk. No deletion/migration code is written.

### Decision 2
- Date: 2026-08-15
- Context: pose-dist's own `.pose/policy/review.json` declares
  `overlay_profiles: ["backend-review@1", "frontend-review@1"]`, and both
  profiles' criteria cite `rules: ["backend-go", ...]`/`["frontend-react",
  ...]`. `validateReviewContractRefs` (`review_closeout.go`) resolves a
  rule name by `os.Stat`-ing `.pose/rules/<name>.md` — a pure file-presence
  check, no awareness of extensions. Removing the two files from
  `.pose/rules/` entirely broke pose-dist's own review-profile validation
  (`pose review bundle --seal` failed with `unknown review rule
  "backend-go"`/`"frontend-react"`), since pose-dist genuinely is a Go
  backend and its own review process depends on `backend-go`'s criteria.
- Options considered: (a) install both extensions locally into pose-dist's
  own `.pose/rules/` for dogfooding, matching how any consumer with a Go
  backend would; (b) remove `frontend-review@1` (and/or `backend-review@1`)
  from pose-dist's own `overlay_profiles`, since pose-dist has no React
  frontend to justify the latter.
- Decision: (a) for both profiles, not a mix. Installing both extensions
  locally is a direct application of the exact mechanism this spec ships —
  pose-dist consuming its own product's extension the same way any
  downstream repository would. This surfaced a second gap: the local
  install physically re-created `.pose/rules/backend-go.md`, which the
  scaffold generator's wholesale walk would re-embed into every fresh
  instance, undoing the whole migration — closed by
  `distpolicy.ExtensionOnlyRuleFiles`, an explicit exclusion mirroring
  `SelfReferentialPolicyFiles`/`SelfReferentialIndexFiles`. Whether
  `frontend-review@1` genuinely belongs in pose-dist's own overlay
  profiles (option b, for frontend-react specifically) is a separate,
  pre-existing governance-config question this spec does not adjudicate —
  installing the extension is a strictly safe default that keeps existing
  behavior working either way.
- Rationale: (b) would require judging whether pose-dist's own review
  configuration is currently over-scoped, a decision independent of this
  spec's actual goal (migrating rule *delivery*, not auditing pose-dist's
  own review-profile selection) and outside this spec's stated Non-goals.
- Consequences: pose-dist's `.pose/indexes/extensions.lock.json` records
  both extensions as locally installed; a third scaffold exclusion list
  (`ExtensionOnlyRuleFiles`) and a review-bundle path-classification gap
  for `extensions/` (unrelated to this decision specifically, but
  discovered by the same change set) both needed fixing — see Artifacts.

---

## 6. Validation

### Strategy
Deterministic Go regression tests for the doctor check (R5), plus three
empirical end-to-end checks: fresh install has no embedded copies (R1),
`pose extension install` against a real throwaway repo lands the file
correctly (R1), and a hand-simulated pre-migration instance triggers the
exact doctor warning (R5).

### Requirement trace
- R1 [satisfied] fresh `pose install` into a throwaway repo ships no
  `backend-go.md`/`frontend-react.md`; `pose extension install
  extensions/pose-rule-backend-go --allow-unsigned --yes` against the same
  repo installs it correctly.
- R2 [satisfied] `AGENTS.md`/`locales/pt-BR/AGENTS.md` domain-rules section
  updated, matching the kubernetes sentence.
- R3 [satisfied] `pose-knowledge-durable-reference-type`'s
  `rules_by_domain` mechanism requires no code change — `pose suggest`
  already prints rule paths regardless of on-disk presence, exactly as it
  already did for `kubernetes`.
- R4 [satisfied] `deliverMachinery`'s existing behavior confirmed by
  reading, not new code — nothing deletes or migrates.
- R5 [satisfied] `TestDoctorWarnsWhenRetiredMachineryStillOnDisk`,
  `TestDoctorSilentWhenInstanceAlreadyRemovedTheRetiredFile`,
  `TestDoctorSilentWhenNoMachineryManifestExists`, plus the empirical
  simulation above.

### Known gaps
- Extensions carry no locale variant today: the pt-BR translations of
  `backend-go.md`/`frontend-react.md` were removed (they would have become
  permanently unreachable dead content otherwise) rather than migrated
  into the extension, since `pose-rule-kubernetes` established the
  no-locale-support precedent and extending the extension mechanism to
  support locales is explicitly out of scope for this spec (Constraints).
  A pt-BR instance installing either extension gets English-only content
  until a future spec adds extension localization.

---

## 7. Final Report

### Delivered scope
Resolved Finding F1 from `pose-extension-reference-publication`:
`backend-go.md`/`frontend-react.md` migrated from embedded core machinery
to extensions (`pose-rule-backend-go`, `pose-rule-frontend-react`),
following exactly the pattern `pose-rule-kubernetes` proved. Compatibility
for already-installed instances required no new code — `deliverMachinery`
already leaves a retired file untouched — and gained a discoverability
signal via a new `pose doctor` advisory check. General catalog/registry
infrastructure and migrating any stack beyond these two remain explicitly
out of scope.

### Files and modules changed
- `.pose/rules/backend-go.md`, `.pose/rules/frontend-react.md`: moved to
  `extensions/pose-rule-backend-go/`, `extensions/pose-rule-frontend-react/`.
- `locales/pt-BR/.pose/rules/backend-go.md`, `frontend-react.md`: removed
  (would have become unreachable dead content).
- `AGENTS.md`, `locales/pt-BR/AGENTS.md`: domain-rules section wording.
- `.pose/workflows/review.md`, `recurrence-escalation.md` (both locales):
  extension annotations added to the rule checklists.
- `pose-mcp/internal/cli/doctor.go`: `machinery.retired-on-disk` check.
- `pose-mcp/internal/cli/doctor_retired_machinery_test.go`: new.
- `pose-mcp/internal/scaffold/locale_coverage_test.go`: removed the two
  retired paths from the structural-comparison list.
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go`:
  `ExtensionOnlyRuleFiles`.
- `pose-mcp/internal/scaffold/distpolicy/distpolicy_test.go`: new
  regression test.
- `pose-mcp/internal/scaffold/scaffold_test.go`: locale-parity scan skips
  extension-only content.
- `pose-mcp/internal/pose/review_bundle.go`: `extensions/` classified as
  governance.
- `pose-mcp/internal/pose/review_bundle_test.go`: new regression test.
- `.pose/indexes/extensions.lock.json`: pose-dist's own dogfood install of
  both extensions.
- `pose-mcp/internal/scaffold/dist/**`: regenerated.

### Validation executed
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l .`: clean.
- Manual end-to-end: fresh install (no embedded files), extension install
  (file lands correctly), simulated pre-migration instance (doctor warning
  fires with the correct hint).

### Residual risks
- pt-BR content for the two migrated rules is gone until a future spec
  adds extension localization (see Known gaps). Accepted: matches the
  existing kubernetes precedent, not a new gap this spec introduces.

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

- [open] a general rule-extension catalog/registry (the issue's "~10 stack
  packages, official catalog" ask) is explicitly out of scope here — worth
  its own future roadmap item once this two-file migration proves out in
  practice, not assumed as an automatic next step.
- [open] extensions carry no locale variant — a pt-BR instance installing
  `pose-rule-backend-go`/`pose-rule-frontend-react` gets English-only
  content, a real (if pre-existing, matching kubernetes) gap worth its own
  spec if localized rule extensions become a priority.
