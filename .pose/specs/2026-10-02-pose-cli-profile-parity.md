---
slug: pose-cli-profile-parity
status: in-progress
created_at: 2026-10-02
completed_at:
priority: 3
components: pose-mcp
task_type: bugfix
delivers: surface:cli-profile-parity
---
# Spec: Preserve CLI facts across profiles and locales

## 1. Intent
### Goal
Finish bounded rendering gaps without changing structured output or forcing migration of untouched legacy commands.
### Business value
Coloured table headers affect column widths, UTF-8 prose wraps by bytes, and pt-BR unknown-token errors retain English token kinds.
### Constraints
Standard library only. Preserve verdict/field contract lines, explicit forced-colour semantics and JSON schema.
### Non-goals
Emptying all legacy print sites, rewriting command messages globally, or changing exit codes.

## 2. Requirements
### Functional
- R1: Coloured table headers shall preserve the alignment of the plain profile.
- R2: Wrap UTF-8 prose by characters without splitting a word; contract fields and verdict lines stay intact.
- R3: Unknown flag and command errors shall use complete en/pt-BR catalog entries, including their suggestion.
- R4: Pin the shared lifecycle renderer under plain, ASCII, Unicode, coloured, narrow, quiet and mixed-stream profiles in both locales; result facts and JSON remain present in all applicable profiles.
### Security
Never put decoration into JSON; automatic colour detection stays per stream.
### Compatibility
Human presentation fixes only; forced colour and machine document contracts unchanged.

## 3. Technical Plan
### Artifacts
- modified: pose-mcp/internal/cli/cliout/render.go
- modified: pose-mcp/internal/cli/cliout/catalog.go
- created: pose-mcp/internal/cli/cliout/profile_contract_test.go
- created: pose-mcp/internal/cli/cliout/testdata/profile-contract.json
- created: .pose/changelogs/unreleased/pose-cli-profile-parity.md
### Delivery targets
- surface:cli-profile-parity module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go
### API/contract changes
None.
### Technical risks
Keep field bytes unchanged; tests compare coloured and plain alignment after stripping ANSI, and JSON independently of decoration.

## 4. Tasks
### Implementation
- [x] Fix alignment, wrapping and token-kind localisation.
### Validation
- [x] Verify profile golden suite, boundary regressions and module matrix.

## 5. Decisions
### Decision 1
- Date: 2026-10-02
- Decision: retain the renderer's existing forced-colour override and preserve legacy call-site ratchet.
- Rationale: the adopted output ADR and shipped documentation explicitly allow forced colour; an unsolicited global migration would exceed this bounded repair.
- Knowledge: knowledge:cli-output-design-taxonomy

## 6. Validation
### Strategy
Test the same semantic finding/field/verdict sequence in every profile. Compare JSON facts, localisation, channel isolation, narrow UTF-8 widths and coloured/plain table alignment. Keep existing machine-channel regressions.
### Deterministic checks
- Command: go -C pose-mcp test ./internal/cli/cliout ./internal/cli -count=1
- Scope: rendering and machine channels
- Expected: exact fixtures, no lost fields or machine colour
### Execution log
Four boundary/profile tests and the entire CLI suite passed. The canonical module matrix passed 50/50 on 2026-10-02; fourteen profile/locale golden cases preserve finding, field and verdict facts.
### Requirement trace
- R1: check:cli-profile-parity-integration — TestColouredTablesKeepPlainAlignment compares stripped coloured and plain output.
- R2: check:cli-profile-parity-integration — TestUTF8ProseWrapsByCharacters covers multibyte prose and indivisible contract fields.
- R3: check:cli-profile-parity-integration — TestUnknownKindsAreFullyLocalised checks flag/command messages and suggestions in both locales.
- R4: check:cli-profile-parity-integration — TestLifecycleRendererProfileGoldens pins seven profiles in en and pt-BR, including quiet facts, isolated streams and valid undecorated JSON.
### Known gaps
Untouched legacy call sites retain their enforced non-growing allowlist; localising those is explicitly outside the parent rendering scope.

## 7. Final Report
### Delivered scope
Coloured alignment, character-aware wrapping and unknown-token localisation are corrected. The shared renderer has fourteen semantic/profile golden cases; existing machine-channel and legacy ratchet behavior is preserved.
### Follow-ups
None.
