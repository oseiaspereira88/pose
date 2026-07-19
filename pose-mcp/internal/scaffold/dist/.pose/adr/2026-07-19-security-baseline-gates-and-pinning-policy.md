# ADR: Security baseline gates and pinning policy

## Status
Accepted (2026-07-19) — spec `pose-ossf-security-baseline`

## Context

Before signing releases can raise trust (supply-chain-trust roadmap,
milestones 2–3), the pipeline itself must resist common compromise paths.
The repository had CI tests but no static analysis, no dependency review, no
secret scanning, no Scorecard measurement, unpinned third-party actions and
one workflow with broader-than-needed permissions.
[OpenSSF Scorecard](https://scorecard.dev/) and the
[GitHub Actions hardening guide](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)
are the benchmark set.

Alternatives considered for scanners: multiple overlapping SAST tools were
rejected (the spec forbids duplicate coverage); the set below covers static
analysis (CodeQL), known vulnerabilities (govulncheck, Go-native and
low-noise), secrets (gitleaks over full history) and dependency diffs
(GitHub dependency review) without redundancy.

For action pinning: pin-everything-by-SHA (including `actions/*`) maximizes
Scorecard score but couples every platform action bump to manual SHA
management; tags-everywhere fails hardening guidance for third parties.

## Decision

- **Scanner set** (`.github/workflows/security.yml`, PR + main + weekly):
  CodeQL (Go, manual build over both modules), `govulncheck@v1.1.4`,
  `gitleaks@v8.21.2` (full history, redacted output) and
  `actions/dependency-review-action` failing on high severity. Go-module
  scanners are invoked with `go run` at pinned versions — authenticated by
  the Go checksum database rather than a mutable action tag.
- **Scorecard** (`.github/workflows/scorecard.yml`): weekly + on-main,
  results published and uploaded as SARIF. The score is a baseline and
  backlog input; the non-goal in the spec stands — it is never presented as
  a guarantee.
- **Pinning policy:** third-party actions are pinned to full commit SHAs
  (goreleaser `e435ccd…`, scorecard `4eaacf0…` — verified against the GitHub
  API at adoption time). First-party `actions/*` and `github/*` remain
  tag-pinned under the owned, expiring exception
  `first-party-actions-tag-pinning` in `.github/security-exceptions.json`.
- **Automatic enforcement (R2):** `TestWorkflowSecurityContract`
  (`internal/version/workflow_security_test.go`) fails when a workflow lacks
  a top-level `permissions` block, when a third-party action is not
  SHA-pinned, or when a referenced exception is missing, incomplete or
  expired. Expiry is a hard failure — exceptions cannot silently outlive
  their review.
- **Release-decision input (R3):** the release workflow re-runs the
  workflow-contract, vulnerability and secret gates before GoReleaser
  publishes. An unwaived critical finding blocks the release; waivers carry
  owner, justification and expiry in the exceptions file (documented in
  `SECURITY.md`).
- **Least privilege:** every workflow declares top-level `contents: read`
  (or `read-all`); elevated scopes (`security-events: write`,
  `pages: write`, `id-token: write`) are granted per job only. Fork PRs run
  read-only jobs; no write token is exposed to PR-triggered gates.

## Consequences

- Positive: the four preventable gap classes (SAST, dependencies, secrets,
  workflow hygiene) now block PRs and releases with owned, expiring waivers
  instead of ad-hoc judgment.
- Positive: workflow hygiene enforcement is a local deterministic test, not
  an external service opinion.
- Trade-off: SHA pins require manual bumps for third-party actions; that
  cost is bounded (two actions today) and is the hardening guidance's intent.
- Residual: CodeQL, govulncheck, gitleaks and Scorecard need network and run
  only in CI; the local gate covers the workflow contract. Scanner noise
  remains a watched risk — the exception file's ownership rule exists to
  prevent broad quiet waivers.
