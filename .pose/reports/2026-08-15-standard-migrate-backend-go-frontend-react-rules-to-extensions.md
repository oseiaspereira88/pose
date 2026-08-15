# POSE Report - 2026-08-15

## Report Type
- standard

## Task
- migrate backend-go/frontend-react rules to extensions
- Task slug: migrate-backend-go-frontend-react-rules-to-extensions
- Spec: pose-domain-rule-extension-migration
- Workflow: feature

## Outcome
- Outcome: pass (source: manual)

## Rules Applied
- security

## Files Changed
- _No files detected_

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-066c09dad60d
- Selector: range:a68724b720dd98d772a066434ca9836c248dd370..b6f6f3de79a62915d9e30172577b2a3fdff74cf5
- Base: a68724b720dd98d772a066434ca9836c248dd370 (a68724b720dd98d772a066434ca9836c248dd370)
- Head: b6f6f3de79a62915d9e30172577b2a3fdff74cf5 (b6f6f3de79a62915d9e30172577b2a3fdff74cf5)
- Diff digest: sha256:5b1965aab1eeeb94925d2c7d76a7fdf777e491a36e9351c090b76f0659e243f2
- Paths:
  - created: .pose/adr/2026-08-15-retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update.md
  - created: .pose/changelogs/unreleased/pose-domain-rule-extension-migration.md
  - created: extensions/pose-rule-backend-go/extension.json
  - created: extensions/pose-rule-frontend-react/extension.json
  - created: pose-mcp/internal/cli/doctor_retired_machinery_test.go
  - modified: .pose/specs/pose-domain-rule-extension-migration/spec.md
  - modified: .pose/workflows/recurrence-escalation.md
  - modified: .pose/workflows/review.md
  - modified: AGENTS.md
  - modified: locales/pt-BR/.pose/workflows/recurrence-escalation.md
  - modified: locales/pt-BR/.pose/workflows/review.md
  - modified: locales/pt-BR/AGENTS.md
  - modified: pose-mcp/internal/cli/doctor.go
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/recurrence-escalation.md
  - modified: pose-mcp/internal/scaffold/dist/.pose/workflows/review.md
  - modified: pose-mcp/internal/scaffold/dist/AGENTS.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/recurrence-escalation.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/review.md
  - modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md
  - modified: pose-mcp/internal/scaffold/locale_coverage_test.go
  - removed: locales/pt-BR/.pose/rules/backend-go.md
  - removed: locales/pt-BR/.pose/rules/frontend-react.md
  - removed: pose-mcp/internal/scaffold/dist/.pose/rules/backend-go.md
  - removed: pose-mcp/internal/scaffold/dist/.pose/rules/frontend-react.md
  - removed: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/backend-go.md
  - removed: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/frontend-react.md
  - renamed: .pose/rules/backend-go.md -> extensions/pose-rule-backend-go/files/.pose/rules/backend-go.md
  - renamed: .pose/rules/frontend-react.md -> extensions/pose-rule-frontend-react/files/.pose/rules/frontend-react.md

## Execution Metadata
- Generated at (UTC): 2026-08-15T12:00:36Z
- Context: feature
- Validation profile: not-provided
- Sequence for task/spec: 1
- Stable comparison hash: 9c61310e49b1b093e3d1b0e5d5cff4fd3b97e29507bc8c06ffd4f7a6369218b5

## Historical Comparison
- Previous execution: _No previous execution_
- Status: first-run
- Stable field diffs:
- _No changes in stable fields_

## Risks
- _No risks provided_

## Follow-ups
- _Add next steps if needed._

## Human Review Needed
- [ ] Review functional impact
- [ ] Review validation coverage
- [ ] Approve merge
