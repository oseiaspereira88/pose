---
type: decision-log
slug: install-one-liner-safe-form
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-13
last_reviewed_at: 2026-08-13
expires_at: 2026-09-12
source_refs:
  spec: "pose-public-install-contract, pose-localization-docs-contract, pose-agent-skills-conformance"
  workflow: "bugfix"
  commands: [go -C pose-mcp test ./... -count=1, pose check --strict, pose validate --tolerant --module pose-mcp]
  external_sources: []
---

# decision-log: install-one-liner-safe-form

## Context

CI on main went red after 65e6097 ("restore one-line POSE installer") with two
failures that shared one root cause class — docs and template drifted from the
deterministic gates:

1. `TestDocsHaveNoUnsafeOrSecretShapedExamples` (security scan reusing the
   `pose-agent-skills-conformance` unsafe patterns) forbids `curl … | bash` in
   README.md and docs-site/docs/*.md, while `TestPublicInstallContract`
   (spec pose-public-install-contract) pinned the exact pipe string as a
   fixture. The two gates were mutually unsatisfiable with the pipe form.
2. `TestEmbeddedDistMatchesPoseDist` flagged POSE.md drift: b991b87 replaced
   the `{{PROJECT_NAME}}` placeholder in the root POSE.md with the literal
   "pose-dist". The root manual doubles as the embedded scaffold template, so
   the literal broke instance-name preservation on refresh
   (`TestRefreshManagedDocsUpdatesAnInstalledManual`) and the pt-BR overlay
   parity (the overlay correctly kept the placeholder).

## Decision

Adopt the safe one-liner as the canonical fast path everywhere:
`curl -fsSLO …/install.sh && bash install.sh` (download then execute, no
pipe). Update the contract-test fixture to the same string so the install
contract now pins the safe form. Restore `{{PROJECT_NAME}}` in the root
POSE.md and regenerate the embedded scaffold.

Rationale: keeps the one-line UX that 65e6097 restored, satisfies the
security gate by construction, and preserves the template/instance duality
the scaffold pipeline depends on. Security precedence (most restrictive rule
wins) over copy-paste convenience.

## Current state

Done: README.md, README.pt-BR.md, docs-site index/quickstart/package-channels,
install.sh header and contract_test.go use the safe form; POSE.md placeholder
restored; `go generate ./internal/scaffold` re-run (103 files synced, no
drift); full `go -C pose-mcp test ./... -count=1` green; `pose check --strict`
and `pose validate --tolerant --module pose-mcp` SUCCESS.

## Next checks

- CI on main after push: `go -C pose-mcp test ./... -count=1` must be green.

## Risks

- External copies of the old pipe one-liner (gists, older docs) remain out of
  repo control; the repo's own surfaces are now uniform.
- A future author may re-introduce `curl | bash` in docs — the security scan
  and the contract fixture now detect both regression directions.
- Root POSE.md must keep `{{PROJECT_NAME}}` while it doubles as the embedded
  template; if a rendered instance manual is ever desired in-repo, split the
  template from the instance file instead of filling the placeholder.

## Next owner

@pose-maintainers

## References

- Commits: 65e6097 (restored one-liner), b991b87 (placeholder replaced),
  d47e38b (security scan introduced).
- Tests: pose-mcp/internal/cli/localization_docs_test.go,
  pose-mcp/internal/version/contract_test.go,
  pose-mcp/internal/scaffold/scaffold_test.go,
  pose-mcp/internal/cli/managed_docs_test.go.
