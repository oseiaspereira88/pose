---
slug: pose-localization-docs-contract
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-public-install-contract, pose-release-compatibility-matrix
priority: 29
---

# Spec: Localization and documentation contract

## 1. Intent

### Goal
keep locales, command examples and public promises synchronized with release behavior.
### Business value
Turns product polish into a release-tested contract.
### Constraints
- Safety and compatibility text cannot ship partially translated.
### Non-goals
- Automatically translate unsupported community content.

## 2. Requirements

### Functional
- R1: Every documented command shall run or parse against the candidate binary.
- R2: Locale key parity and fallback shall be tested for CLI and scaffold.
- R3: Docs shall separate tutorial, how-to, reference and explanation with version applicability.

### Non-functional
- Build docs with strict links and deterministic snippets.

### Security
- Scan examples for secrets, unsafe downloads and permissions.

### Compatibility
- Preserve stable anchors and redirect moved pages where practical.

## 3. Technical Plan

### Affected areas
- MkDocs, README, locales, CLI help, scaffold and docs CI.

### API/contract changes
- Define locale completeness and docs compatibility policy.

### Data/storage changes
- Maintain snippet fixtures and translation key inventories.

### Technical risks
- Generated snippets can hide editorial context.

### Primary references
- [Diátaxis](https://diataxis.fr/)
- [Unicode CLDR](https://cldr.unicode.org/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Diátaxis](https://diataxis.fr/).

### Implementation
- [ ] Classify pages and define locale/release gates. ([reference](https://diataxis.fr/))
- [ ] Execute snippets and validate links, anchors and fallback keys. ([reference](https://cldr.unicode.org/))
- [ ] Review rendered security-critical and compatibility text. ([reference](https://diataxis.fr/))

### Validation
- [ ] Run `pose check --strict && mkdocs build --strict -f docs-site/mkdocs.yml` and retain evidence. ([reference](https://diataxis.fr/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://cldr.unicode.org/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-localization-docs-contract-self-inspecting-tests.md` (Accepted): unified the locale overlay to one path convention across templates/workflows/rules/skills (fixing a real asymmetry that let a locale-parity bug ship); extended the existing scaffold parity test's prefix list rather than special-casing it; derived the documented-commands contract test from `cli.go`'s own switch statement instead of a hand-maintained duplicate list; added a visible Diátaxis type + version-applicability line to every docs page; reused the existing skills-conformance security patterns against docs content. Rejected: special-casing the parity test's path mapping for templates only (preserves the root asymmetry); a hand-maintained "valid commands" list for the R1 test (a second copy of the dispatch table that can itself drift).

## 6. Validation

**Strategy:** validate the locale-parity fix at both the scaffold-source level and the install-time end-to-end level, derive the documented-commands contract from the live CLI dispatch table, verify Diátaxis classification and version applicability on every page, and scan docs for secret-shaped/unsafe-command/permissions patterns using the existing skills-conformance scanner.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... ./internal/scaffold/... -run 'Locale|Diataxis|DocumentedCommands|DocsHaveNoUnsafe|EditorialDefaults' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-localization-docs-contract --ready-check`.

### Requirement trace
- R1 [satisfied] every `pose <command>` mention in `README.md` and `docs-site/docs/*.md` is checked against the command set derived from `cli.go`'s live switch statement; check:test (TestDocumentedCommandsAreRecognizedByTheCLI)
- R2 [satisfied] locale overlay unified to one path convention (templates/workflows/rules/skills all mirror the English path exactly under `locales/<locale>/`); the pre-existing scaffold parity test now covers `.pose/templates/` (previously excluded, the actual root cause); install-time end-to-end proof that all five templates localize, not just the two a prior hand-picked assertion checked; unsupported-locale fallback proven to leave zero non-English content; check:test (TestEditorialDefaultsAreEnglishAndPtBROverlayIsComplete in internal/scaffold, the extended locale assertion in TestNativeScaffoldsCreateContractArtifacts, TestInstallFallsBackToEnglishWhenLocaleUnsupported)
- R3 [satisfied] every docs-site page carries a visible `**Doc type:** <Tutorial|How-to|Reference|Explanation> · **Applies to:** POSE ≥ 0.9.0` line; check:test (TestDocsAreDiataxisClassifiedWithVersionApplicability)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... ./internal/scaffold/... -v -count=1` (relevant subset: Locale, Diataxis, DocumentedCommands, DocsHaveNoUnsafe, EditorialDefaults, plus the extended `TestNativeScaffoldsCreateContractArtifacts`) — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-localization-docs-contract --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Security (scan examples for secrets, unsafe downloads, permissions): TestDocsHaveNoUnsafeOrSecretShapedExamples reuses `unsafeSkillPatterns` (curl|sh, wget|sh, rm -rf /, --no-verify, disable TLS verification) and `secretLikePatterns` against every doc, plus a new `sudo`-in-example check — all pass with zero matches across README.md and all 12 docs-site pages.
- Non-functional (strict links, deterministic snippets): `mkdocs build --strict -f docs-site/mkdocs.yml` is not executable in this sandbox (no `pip`/`mkdocs` available); already wired in `.github/workflows/docs.yml` on every PR touching `docs-site/**` — deferred to that CI run, consistent with how `pose-package-manager-distribution` and `pose-slsa-provenance` handled sandbox-unavailable infrastructure. Follow-up opened to confirm the first run against this spec's page edits.
- Compatibility (preserve stable anchors, redirect moved pages): satisfied by construction — every edit in this spec inserts a line after an existing H1; no heading, anchor, or page was renamed or moved, so no redirect is needed.

## 7. Final Report

- Delivered scope: fixed a real, previously undetected locale-parity bug (English default templates that were actually Portuguese, with no `pt-BR` translation on file) by unifying `install.go`'s locale-overlay path convention and extending the parity test that should have caught it; added a self-inspecting documented-commands contract test (R1) that reads the CLI's own dispatch table instead of duplicating it; classified all 12 docs-site pages by Diátaxis type with a visible version-applicability line (R3); added a docs security scan reusing the existing skills-conformance patterns.
- Residual risk: generated snippets can still hide editorial context if a future doc page is added without running the Diátaxis/security tests locally before commit — mitigated by both tests running in the same `go test ./...` every contributor and CI already run, not a separate opt-in step.
- Follow-ups: see below.

### Follow-ups

- [open] Confirm the first `mkdocs build --strict` CI run (`docs.yml`) against this spec's page edits — not executable in this sandbox (no `pip`/`mkdocs`). (owner:@pose-maintainers crit:medium review:2026-08-19)
