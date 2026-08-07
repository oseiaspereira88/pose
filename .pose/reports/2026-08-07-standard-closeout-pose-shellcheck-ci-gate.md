# POSE Report - 2026-08-07

## Report Type
- standard

## Task
- closeout pose-shellcheck-ci-gate
- Task slug: closeout-pose-shellcheck-ci-gate
- Spec: pose-shellcheck-ci-gate

## Outcome
- Outcome: unknown (source: manual)

## Rules Applied
- _Not provided_

## Files Changed
- pose/indexes/delivery-integrity.json
- .pose/indexes/releases.json
- .pose/indexes/spec-graph.json
- .pose/state/history.jsonl
- .pose/state/project-state.md
- .pose/state/refresh-log.jsonl
- pose-mcp/internal/scaffold/dist/.pose/indexes/delivery-integrity.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/releases.json
- pose-mcp/internal/scaffold/dist/.pose/indexes/spec-graph.json
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-cyclonedx-sbom/spec.md
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-first-release-evidence-confirmation/spec.md
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-governance-gate-activation/spec.md
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-package-manager-distribution/spec.md
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-release-signing/spec.md
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-verifier-assets-variable-fix/spec.md
- pose-mcp/internal/scaffold/dist/scripts/release.sh
- .pose/reports/2026-08-07-standard-closeout-pose-sbom-license-inventory.md
- .pose/reports/history/standard-closeout-pose-sbom-license-inventory.jsonl
- pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-package-channel-delivery.md
- pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-release-signing-rejection.md
- pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-release-workflow-hardening.md
- pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-sbom-license-inventory.md
- pose-mcp/internal/scaffold/dist/.pose/changelogs/unreleased/pose-shellcheck-ci-gate.md
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-package-channel-delivery/
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-release-signing-rejection/
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-release-workflow-hardening/
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-sbom-license-inventory/
- pose-mcp/internal/scaffold/dist/.pose/specs/pose-shellcheck-ci-gate/

## Validation Commands
- _Fill manually_

## Results
- _No validation output detected_

## Change Set
- ID: cs-faf157445450
- Selector: range:6dd5b86..e3cc340
- Base: 6dd5b86 (6dd5b860a22e75206b4c1412369d0becee6b06ba)
- Head: e3cc340 (e3cc340d8f0adff13bbe0750eb8f459dcf248e67)
- Diff digest: sha256:8713b937cfe7c6fa21ccf62dd516b69c802a130a11e17b29e7eb9fa50966aa96
- Paths:
  - created: .pose/changelogs/unreleased/pose-shellcheck-ci-gate.md
  - created: .pose/specs/pose-shellcheck-ci-gate/spec.md
  - modified: .github/workflows/ci.yml
  - modified: .pose/specs/pose-verifier-assets-variable-fix/spec.md
  - modified: scripts/release.sh

## Execution Metadata
- Generated at (UTC): 2026-08-07T20:25:07Z
- Context: closeout
- Validation profile: strict
- Sequence for task/spec: 1
- Stable comparison hash: 849f4d4bfa9345b4edc9c76d21404a7e240e8f6a590816bf53e082e9c229d998

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
