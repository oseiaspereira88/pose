---
slug: pose-rule-extension-locale-parity
status: done
created_at: 2026-08-15
completed_at: 2026-08-15
supersedes:          # slug of the superseded spec (when applicable)
depends_on:
priority: 3
components: pose-mcp
delivers: capability:rule-extension-locale-parity
---

# Spec: pose-rule-extension-locale-parity

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
Give `pose-rule-backend-go` and `pose-rule-frontend-react` (the two domain
rule extensions `pose-domain-rule-extension-migration` shipped in v1.3.0) a
pt-BR content variant, so a pt-BR instance installing either receives
locale-consistent content instead of English-only rule text alongside
pt-BR core machinery.

### Business value
Named as a follow-up in `pose-domain-rule-extension-migration`'s Final
Report: "extensions carry no locale variant — a pt-BR instance installing
`pose-rule-backend-go`/`pose-rule-frontend-react` gets English-only
content, a real (if pre-existing, matching kubernetes) gap." This repository's
own governance documents (`README.pt-BR.md`) and `pose-locale-coverage-contract`
(a dedicated gate enforcing English/pt-BR parity for core machinery) treat
pt-BR as a first-class, contractually-covered locale — extensions are the
one distribution path that gate does not reach today, so the parity
guarantee the project already makes has a silent exception.

### Constraints
- Must follow whatever locale-selection mechanism core machinery already
  uses (`--locale`, brownfield auto-detection via `machineryLocale()`) —
  not a parallel, extension-specific mechanism.
- `pose-rule-kubernetes` has the identical gap and predates this work;
  confirm during Technical Plan whether this spec's mechanism should also
  close it or whether that is explicitly out of scope (kubernetes is a
  third extension, not one of the two this roadmap's predecessor shipped).

### Non-goals
- Localizing extension content into any locale beyond pt-BR — no other
  locale exists in this project yet (`pose-locale-coverage-contract`'s own
  Known gaps note this: "Only pt-BR exists, so the contract has never been
  exercised by a second locale").
- Building a general N-locale extension-authoring framework ahead of a
  second real locale actually existing.

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
- R1: `pose extension install` shall accept an explicit `--locale <tag>`
  flag; when the package ships a `locales/<tag>/files/<target>` variant of
  a declared target, that content shall be installed instead of the
  package's base content.
- R2: when `--locale` is omitted, `pose extension install` shall
  auto-detect the target repository's locale the same way `pose install`/
  `pose update` already do (`machineryLocale`), and use it exactly as if
  it had been passed explicitly.
- R3: `pose-rule-backend-go` and `pose-rule-frontend-react` shall ship a
  `locales/pt-BR/files/` variant of their one rule file each, using the
  exact prior pt-BR translation this repository already carried (removed
  by `pose-domain-rule-extension-migration` when the file was deleted from
  `locales/pt-BR/.pose/rules/`, recovered from git history here) rather
  than a fresh, unverified translation.
- R4: a package with no matching `locales/<tag>/` variant shall install
  its base content unchanged — no regression for `pose-rule-kubernetes` or
  any future extension that never ships a locale overlay.

### Non-functional
- No network calls, no generative translation — content is either
  excerpted verbatim from this repository's own prior pt-BR translation
  (R3) or the existing base content (R4).

### Security
- Locale resolution does not change what `verifyExtensionSignature`
  verifies (`extension.json` only, unchanged) or which target paths a
  package may write (`extensionWhitelist`/`m.Permissions`, unchanged) — it
  only changes which package-relative source path is read for an already-
  validated target.

### Compatibility
- `pose extension verify`'s reported digest stays locale-independent (the
  base `files/` content) — a locale overlay is additional content the
  package ships, not an alternate "version" needing its own identity.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/extension.go` (`cmdExtensionInstall`,
  `applyExtensionInstall`, new `localizedExtensionSource`)
- `extensions/pose-rule-backend-go/`, `extensions/pose-rule-frontend-react/`
  (new `locales/pt-BR/files/` overlays)

### Artifacts
- modified: pose-mcp/internal/cli/extension.go
- created: pose-mcp/internal/cli/extension_locale_test.go
- created: extensions/pose-rule-backend-go/locales/pt-BR/files/.pose/rules/backend-go.md
- created: extensions/pose-rule-frontend-react/locales/pt-BR/files/.pose/rules/frontend-react.md

### Delivery targets
- capability:rule-extension-locale-parity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- `pose extension install` gains one new optional flag, `--locale <tag>` —
  additive, existing invocations without it behave identically (R4).

### Technical risks
- Low: `localizedExtensionSource` is a pure path-resolution function with a
  simple existence check; falls back to today's exact behavior whenever no
  overlay exists (R4), verified with a real (non-stubbed) end-to-end
  `pose extension install` run in a throwaway repository during
  implementation, not only unit tests.

---

## 4. Tasks

### Planning
- [x] Confirmed `pose extension install` has zero locale awareness today —
      a flat `pkgDir/files/<target>` copy, no `--locale` flag
- [x] Confirmed `verifyExtensionSignature` only signs `extension.json`, not
      `files/` content — a locale overlay does not touch the trust
      boundary a signature guarantees (Security)
- [x] Found the exact prior pt-BR translations for both rule files in git
      history (deleted by `pose-domain-rule-extension-migration`'s
      migration commit) and confirmed the current English extension
      content is byte-identical to what those translations were made
      against — reused verbatim instead of re-translating (R3)

### Implementation
- [x] `localizedExtensionSource(pkgDir, locale, target)`: prefers
      `locales/<locale>/files/<target>` when present, else base `files/`
      (R1, R4)
- [x] `cmdExtensionInstall`: `--locale` flag; auto-detects via
      `machineryLocale(scaffold.Dist(), root, "en")` when omitted (R2)
- [x] `applyExtensionInstall` takes the resolved locale, reads from
      `localizedExtensionSource` instead of the hardcoded base path
- [x] Recovered pt-BR overlays for both extensions from git history (R3)
- [x] Decided kubernetes' identical gap stays open (Decision 1)

### Validation
- [x] Unit test for `localizedExtensionSource`'s four cases (overlay
      present, absent, empty locale, `"en"`)
- [x] Integration tests: explicit `--locale pt-BR`, fallback with no
      overlay, auto-detection without the flag
- [x] Real (non-stubbed) end-to-end run: built the dev binary, installed
      pt-BR core into a throwaway repo, then installed both extensions —
      once with `--locale pt-BR` explicit, once relying on auto-detection
      — confirmed genuine pt-BR content installed both times, then cleaned
      up the throwaway repo (no stray state left in this repository)
- [x] `go test ./...`, `go vet ./...`, `gofmt -l .`: all clean

---

## 5. Decisions

> Optional section. Use it when the implementation involves trade-offs or
> alternatives.

### Decision 1
- Date: 2026-08-15
- Context: `pose-rule-kubernetes` has the exact same missing-locale gap
  (Constraints) and the mechanism this spec builds works for any
  extension, not only the two named in Goal.
- Options considered: (a) also add a pt-BR overlay for
  `pose-rule-kubernetes` in this spec; (b) leave it out, mechanism-only for
  that extension.
- Decision: (b).
- Rationale: unlike `backend-go`/`frontend-react`, no prior pt-BR
  translation of `kubernetes.md` exists in this repository's history to
  recover verbatim (R3's exact approach) — a from-scratch translation
  carries real accuracy risk for review-rule content with no second
  reviewer to check Portuguese technical phrasing against. Kubernetes is
  also the one extension actually signed and published through the
  release workflow (`pose-extension-reference-publication`) — adding
  locale content to it touches the packaging/signing step too, a
  materially larger change than this spec's two unpublished extensions.
- Consequences: `pose-rule-kubernetes` keeps installing English-only
  content regardless of target locale, via R4's unchanged fallback — no
  regression, just an unclosed pre-existing gap, tracked as a Follow-up.

### Decision 2
- Date: 2026-08-15
- Context: needed a translation for R3 without introducing a new,
  unverified one.
- Options considered: (a) write a new pt-BR translation now; (b) recover
  the translation `pose-domain-rule-extension-migration` deleted from
  `locales/pt-BR/.pose/rules/{backend-go,frontend-react}.md` when it moved
  the rules out of core machinery.
- Decision: (b), after confirming (via `diff`) the current English
  extension content is byte-identical to the English original that prior
  translation was made against.
- Rationale: reusing a previously-reviewed translation is strictly safer
  than authoring a new one under this session's own judgment, and the
  byte-identity check ruled out translating against stale content.
- Consequences: none — this is the ideal case, not a compromise.

---

## 6. Validation

### Strategy
Unit test for the pure resolution function; three integration tests
covering the explicit flag, the no-overlay fallback and auto-detection,
using the same signature-stubbing pattern (`fakeSignedInstall`) the
existing extension test suite already established; a genuine, non-stubbed
end-to-end run against a real built binary in a throwaway directory,
because the mechanism's whole point is user-visible file content and unit
tests alone would not catch a wiring mistake in `cmdInstall`/
`cmdExtensionInstall` glue.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./...`
- Scope: whole module
- Expected: PASS

#### Lint
- Command: `go -C pose-mcp vet ./...` / `gofmt -l .`
- Scope: whole module
- Expected: clean

#### Build
- Command: `go -C pose-mcp build -trimpath -o ./pose ./cmd/pose`
- Scope: `cmd/pose`
- Expected: builds; used for the manual end-to-end run

### Execution log
- Date: 2026-08-15
- Environment: local (Linux, Go toolchain per `go.mod`); manual end-to-end
  run against the freshly built binary in a throwaway `/tmp` directory,
  removed after the run.
- Notes: both extensions verified with `--locale pt-BR` explicit and with
  auto-detection (no flag) against a real pt-BR-installed target.

### Results summary
- Successes: `go test ./...`, `go vet ./...`, `gofmt -l .` all clean; four
  new tests pass; manual end-to-end run confirmed real pt-BR content
  installed in both modes.
- Failures: none.
- Warnings: none.

### Requirement trace
- R1 [satisfied] test:TestExtensionInstallUsesExplicitLocaleOverlay
- R2 [satisfied] test:TestExtensionInstallAutoDetectsTargetLocaleWithoutFlag
- R3 [satisfied] manual end-to-end run, 2026-08-15 (real pt-BR content installed for both extensions)
- R4 [satisfied] test:TestExtensionInstallFallsBackToBaseWhenLocaleOverlayAbsent

### Known gaps
- `pose-rule-kubernetes` keeps its pre-existing English-only gap
  (Decision 1) — not a regression, but not closed here either.

---

## 7. Final Report

### Delivered scope
`pose extension install` now resolves a package's `locales/<tag>/files/`
overlay — explicitly via `--locale` or auto-detected from the target the
same way core machinery already does. `pose-rule-backend-go` and
`pose-rule-frontend-react` ship their pt-BR variant, recovered verbatim
from this repository's own git history rather than freshly translated.
`pose-rule-kubernetes` deliberately stays out of scope (Decision 1).

### Files and modules changed
- `pose-mcp/internal/cli/extension.go`: `--locale` flag,
  `localizedExtensionSource`, locale-aware `applyExtensionInstall`.
- `pose-mcp/internal/cli/extension_locale_test.go` (new): tests.
- `extensions/pose-rule-backend-go/locales/pt-BR/files/.pose/rules/backend-go.md` (new).
- `extensions/pose-rule-frontend-react/locales/pt-BR/files/.pose/rules/frontend-react.md` (new).

### Validation executed
- Command: `go -C pose-mcp test ./...`
- Result: SUCCESS
- Command: manual end-to-end `pose extension install` (built binary,
  throwaway repo, both explicit and auto-detected locale)
- Result: real pt-BR content installed correctly in both modes

### Residual risks
- None beyond the accepted Known gap (`pose-rule-kubernetes`).

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

- [open] `pose-rule-kubernetes` still has no pt-BR variant (Decision 1) —
  worth its own small follow-up once a reviewed translation exists, given
  it is also the one extension actually signed/published, so closing it
  touches the release-signing step this spec deliberately left untouched.
