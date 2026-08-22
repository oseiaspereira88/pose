---
slug: pose-brownfield-reference-kits
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-standalone-dogfood, pose-monorepo-validation-recipes, pose-agent-skills-conformance
priority: 28
---

# Spec: Brownfield reference kits

## 1. Intent

### Goal
publish executable adoption kits for existing repos using POSE alone and with Spec Kit/OpenSpec.
### Business value
Demonstrates incremental value without demanding a governance rewrite.
### Constraints
- Represent imperfect repositories and keep lifecycle authority explicit.
### Non-goals
- Promise automatic semantic migration without curation.

## 2. Requirements

### Functional
- R1: Kits shall cover direct adoption, Spec Kit import and OpenSpec import/reconciliation.
- R2: Each kit shall progress from visibility to blocking gates with rollback.
- R3: CI shall execute commands and assert preservation, warnings and readiness.

### Non-functional
- Keep kits small, reproducible and release-versioned.

### Security
- Use sanitized fixtures and test symlink, overwrite and boundary rejection.

### Compatibility
- Document mapping loss and retain source provenance.

## 3. Technical Plan

### Affected areas
- Examples, import adapters, docs, CI and extension fixtures.

### API/contract changes
- Publish pathways and authority-transfer rules.

### Data/storage changes
- Version source fixtures, mapping reports and post-adoption snapshots.

### Technical risks
- Idealized examples conceal real migration costs.

### Primary references
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)
- [OpenSpec](https://github.com/Fission-AI/OpenSpec)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md).

### Implementation
- [ ] Design representative greenfield, brownfield and mixed-SDD fixtures. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [ ] Implement staged guides with preservation assertions. ([reference](https://github.com/Fission-AI/OpenSpec))
- [ ] Measure time-to-first-gate and document mapping loss. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Import|Install|Preserve'` and retain evidence. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://github.com/Fission-AI/OpenSpec))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-brownfield-kits-checked-in-fixtures-git-native-rollback.md` (Accepted): three real, checked-in fixtures under `examples/` (excluded from the scaffold embed, same pattern as `tests/`), driven by Go tests calling the exact CLI entry points rather than a new shell script; fixtures are deliberately incomplete (missing `plan.md`/`design.md`) to exercise genuine curation-warning surfacing; rollback documented and proven as plain git revert rather than building a new uninstall command. Rejected: narrative-only docs (drift risk the spec explicitly flags); a new `pose adopt`/`pose uninstall` command pair (unnecessary — nothing pre-existing is ever mutated, so git already gives free rollback).

## 6. Validation

**Strategy:** validate each kit end to end against its real, checked-in fixture — preservation of pre-existing content (byte-for-byte), surfaced curation warnings, DoR readiness of the generated artifact, and git-native rollback safety (zero modification to anything pre-existing).

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'Brownfield' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-brownfield-reference-kits --ready-check`.

### Requirement trace
- R1 [satisfied] three kits cover direct adoption, Spec Kit import and OpenSpec import/reconciliation; check:test (TestBrownfieldDirectAdoptionKit, TestBrownfieldSpecKitImportKit, TestBrownfieldOpenSpecImportKit)
- R2 [satisfied] each kit's README and test progress visibility (dry-run / read-only doctor) → adoption/import → blocking gate (`validate --strict`, `check --strict`, `lint-spec --ready-check`), with rollback documented as a plain git revert and proven via a zero-modification `git status --porcelain` assertion; check:test (all three), check:doc (`examples/brownfield-kits/*/README.md`)
- R3 [satisfied] CI (`go test ./...`) executes every staged command against the real fixture and asserts preservation (byte-for-byte pre-existing content), warnings (the intentionally-missing `plan.md`/`design.md` curation notes), and readiness (`spec.ready=true`); check:test (all three)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'Brownfield' -v -count=1` — SUCCESS (3 tests, each against its real `examples/brownfield-kits/<kit>/fixture` tree).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold` (confirmed `examples/` does not enter the embedded scaffold: file count unchanged from before this spec's fixtures were added).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-brownfield-reference-kits --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Security (sanitized fixtures; symlink/overwrite/boundary rejection): fixtures contain only synthetic placeholder content (`NotImplementedError` stubs, no real credentials or logic); symlink/overwrite/boundary rejection is already covered at the engine level by the pre-existing `TestImportRejectsSymlinkAndMalformedOpenSpec` and `TestImportPreflightCollisionLeavesBatchUntouched` (unchanged by this spec — the kits exercise the happy/curation path, the engine tests exercise the adversarial path).
- Compatibility (document mapping loss, retain source provenance): every generated spec's `## 8. Import Provenance` section (pre-existing importer behavior, unchanged) records format, source path and every consumed artifact; this spec's kits additionally document and assert the curation-warning behavior for a realistic incomplete source in each README.

## 7. Final Report

- Delivered scope: three checked-in, executable adoption kits (`examples/brownfield-kits/{direct-adoption,spec-kit-import,openspec-import}/`) with staged visibility→adoption→blocking-gate READMEs, each verified end to end by a dedicated Go test against its real fixture; `examples/` added to the scaffold's exclusion list; root `README.md` links to the kits from the existing import section.
- Residual risk: idealized examples can still conceal real migration cost at larger scale — mitigated by choosing intentionally-imperfect fixtures (missing companion files) rather than pristine synthetic ones, but these are still small, single-feature examples; a genuinely large legacy spec-kit/OpenSpec tree may surface curation costs these kits don't represent.
- Follow-ups: none — all three requirements are satisfied with executed evidence and no sandbox-unavailable gap (unlike the release-pipeline specs, these kits need no network or external infrastructure to test end to end).
