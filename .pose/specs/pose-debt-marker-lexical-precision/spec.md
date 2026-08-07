---
slug: pose-debt-marker-lexical-precision
status: done
created_at: 2026-08-03
completed_at: 2026-08-03
supersedes:
depends_on: pose-project-agnostic-assessment-engines
priority: 0
components: pose-mcp, harness-governance
delivers: governance:project-assessment
---

# Spec: Technical-debt marker lexical precision

## 1. Intent

### Goal
Prevent ordinary lowercase prose from being classified as declared technical
debt while preserving explicit uppercase comment markers and executable debt
constructs.

### Business value
POSE must work in multilingual repositories. The Portuguese word `todo` and
ordinary explanations containing `stub` cannot create fictitious backlog.

### Constraints
- Preserve schema 1, CLI/MCP contracts and current marker names.
- Apply the same classifier to discovery and technical-debt reports.

### Non-goals
- Introduce language-specific parsers.
- Change coverage reconciliation.

## 2. Requirements

### Functional
- R1: Uppercase `TODO`, `FIXME`, `HACK` and `STUB` in comments remain markers.
- R2: Lowercase natural-language `todo` and `stub` in comments are ignored.
- R3: Executable `panic`, `todo!`, `unimplemented!` and not-implemented
  constructs remain classified independently of comment-marker case.
- R4: Discovery and technical-debt reports use the same corrected classifier.

### Non-functional
- Add a neutral regression fixture with multilingual lowercase prose.

### Security
- No filesystem or path policy change.

### Compatibility
- Output schema and public commands remain unchanged.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/techdebt.go`
- `pose-mcp/internal/pose/assessment_agnostic_test.go`
- generated scaffold and Harness governance mirror.

### Delivery targets
- governance:project-assessment module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### Artifacts
- modified: pose-mcp/internal/pose/techdebt.go
- modified: pose-mcp/internal/pose/assessment_agnostic_test.go
- created: .pose/specs/pose-debt-marker-lexical-precision/spec.md
- created: .pose/changelogs/v0.16.3/pose-debt-marker-lexical-precision.md
- created: .pose/reports/2026-08-03-pose-debt-marker-lexical-precision-review.md

### API/contract changes
- None; only false-positive suppression.

### Data/storage changes
- Regenerated assessments may contain fewer, more accurate debt items.

### Technical risks
- Lowercase intentional markers cease to count; explicit governance markers
  are conventionally uppercase and remain supported.

## 4. Tasks

### Planning
- [x] Reproduce false positives from Harne8 dogfooding.

### Implementation
- [x] Make comment declarations case-sensitive.
- [x] Preserve case-insensitive executable construct detection.
- [x] Add multilingual prose regression.

### Validation
- [x] Pass focused and full Go tests/vet.
- [x] Re-run assessments on Harne8.
- [x] Review and close the patch implementation.
- [x] Publish and independently verify the patch release.

## 5. Decisions

### Decision 1
- Date: 2026-08-03
- Context: natural-language words were indistinguishable from marker tokens.
- Options considered: require a colon; maintain a stop-word list; require
  uppercase declarations.
- Decision: comment markers are case-sensitive uppercase tokens; executable
  constructs keep their language-oriented case-insensitive detection.
- Rationale: deterministic, language-neutral and backward-compatible with the
  established marker convention.
- Consequences: intentional lowercase comment markers must be normalized.

## 6. Validation

### Strategy
Extend the neutral fixture with lowercase Portuguese/English prose and retain
the existing exact total and executable Rust coverage.

### Checks determinísticos
- `go -C pose-mcp test ./internal/pose -run ProjectAgnostic -count=1`.
- `go -C pose-mcp test ./...`.
- `go -C pose-mcp vet ./...`.
- `pose check --strict` and `pose validate`.

### Cenários negativos
- `// todo projeto usa um test stub` produces no item.
- `// TODO: explicit debt` still produces an item.
- `todo!()` still produces a `STUB` item.

### Log de execução
- 2026-08-03: false positives reproduced by the corrected v0.16.2 engine on
  actual Portuguese source comments during the parent-repository review.
- 2026-08-03: neutral regression, full Go suite/vet, strict POSE checks,
  `pose validate` and `govulncheck` passed.
- 2026-08-03: Harne8 dogfooding now reports two real markers instead of nine;
  lowercase prose no longer appears.
- 2026-08-03: `v0.16.3` published by workflow `30832362547` and independently
  verified by workflow `30832724504`; release ledger reached `verified`.

### Resumo de resultados
- Classifier corrected with no public contract change; all implementation,
  review, publication and independent verification gates passed.

### Gaps conhecidos
- None accepted; spec remains open until the replacement patch is verified.

### Requirement trace
- R1 [satisfied] test:TestProjectAgnosticAssessmentEngines.
- R2 [satisfied] test:TestProjectAgnosticAssessmentEngines.
- R3 [satisfied] test:TestProjectAgnosticAssessmentEngines.
- R4 [satisfied] check:shared-debt-lex-state.

## 7. Final Report

### Escopo entregue
Case-sensitive explicit comment declarations and preserved executable debt
detection, with multilingual regression coverage.

### Arquivos e módulos alterados
- `pose-mcp/internal/pose`, generated mirrors and release artifacts.

### Validação executada
- Focused/full Go tests, vet, strict POSE checks, full validate, vulnerability
  scan and real-project dogfooding.

### Riscos residuais
- Static markers remain declarations, not runtime evidence.

### Follow-ups
No follow-up.
