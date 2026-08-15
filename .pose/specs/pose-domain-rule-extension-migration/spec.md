---
slug: pose-domain-rule-extension-migration
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: 2026-08-15
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 2
components: pose-mcp
depends_on:
delivers:
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
- R4: whatever compatibility strategy Decision 1's ADR settles on for
  already-installed instances shall be implemented exactly as decided — not
  substituted for a simpler default at implementation time.

### Non-functional
- No change to the extension mechanism itself (signing, whitelist,
  transactional install) — this spec only produces two new extension
  packages and removes their embedded equivalents, it does not modify
  `extension.go`.

### Compatibility
- **Breaking for already-installed instances** unless Decision 1's ADR
  specifies a migration path — flagged explicitly rather than assumed
  additive. See Decision 1.

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
- `pose-mcp/internal/scaffold/dist/**` (regenerated)

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
- [ ] TBD — depends on Decision 1

### Validation
- [ ] TBD — depends on Decision 1

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
- Decision: not yet made — deferred to a dedicated ADR, per this roadmap's
  `rule-extensionization` milestone risk control ("must not ship until its
  compatibility ADR explicitly states what happens to already-installed
  instances"). This spec cannot move to `in-progress` until that ADR is
  accepted.
- Rationale: this is a structural/contract change (AGENTS.md's own
  definition of when an ADR is required) affecting every already-installed
  instance, not a purely additive change — deciding it inline, without the
  scrutiny an ADR forces, risks silently degrading agents' governance
  context in every downstream repo already running POSE.
- Consequences: this spec stays `draft` with Implementation/Validation
  tasks as placeholders until Decision 1 closes.

---

## 6. Validation

### Strategy
To be defined once Decision 1 resolves: extension packaging/signing
verification for the two new packages, plus a compatibility test exercising
whatever migration path the ADR settles on for already-installed instances.

### Requirement trace
<!-- At closeout, one bullet per declared R-ID. -->

### Known gaps
- Decision 1 is open; this spec cannot close until it resolves.

---

## 7. Final Report

### Delivered scope
<!-- What was implemented and what was intentionally left out. -->

### Files and modules changed
- 

### Validation executed
- Command:
- Result:

### Residual risks
- 

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
