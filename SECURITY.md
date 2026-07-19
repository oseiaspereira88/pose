# Security Policy

## Supported versions

POSE is pre-1.0. Only the latest released version receives security fixes.

## Reporting a vulnerability

**Do not open a public issue for security reports.**

Email **oseiaspereira88@gmail.com** with:

- a description of the vulnerability and its impact;
- reproduction steps or a proof of concept;
- the POSE version / commit hash affected.

You will receive an acknowledgment within **5 business days**. We ask for up to
**90 days** of coordinated disclosure before any public write-up; we will credit
you in the release notes unless you prefer otherwise.

## Supply-chain gates

Every pull request runs CodeQL static analysis, `govulncheck` (known
vulnerabilities), gitleaks secret detection over the full history and GitHub
dependency review (`.github/workflows/security.yml`). OpenSSF Scorecard runs
weekly and on main with published results (`.github/workflows/scorecard.yml`).

Workflow hygiene is a tested contract (`TestWorkflowSecurityContract`): every
workflow declares least-privilege permissions, third-party actions are pinned
to full commit SHAs, and first-party tag pinning is only allowed through an
owned, expiring exception in `.github/security-exceptions.json` — expired
exceptions fail CI until renewed or fixed.

Unresolved critical findings are explicit release-decision inputs: the release
workflow re-runs the vulnerability, secret and workflow-contract gates and
refuses to publish while an unwaived critical finding exists. Waivers live in
the exceptions file with an owner, a justification and an expiry date; a
Scorecard number is a baseline input, never a security guarantee.

## Scope notes

- The POSE engine is designed to run **offline**: gates make no network calls.
  Any gate observed performing network I/O is itself a reportable issue.
- The MCP server (`pose-mcp`) is read-heavy by design; any write path reachable
  through an MCP tool without explicit configuration is a reportable issue.
- Secrets do not belong in `.pose/` artifacts. A template or workflow that
  encourages placing credentials in versioned files is a reportable issue.
