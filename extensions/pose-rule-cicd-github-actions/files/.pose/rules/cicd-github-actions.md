# Rule: CI/CD GitHub Actions

## When to consult

Consult this guide for GitHub Actions workflow files (`.github/workflows/*.yml`), composite actions, reusable workflows, and CI/CD security configurations.

## Required patterns

- Pin third-party actions to full immutable commit SHAs with a comment containing the release tag (e.g. `uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2`).
- Enforce least-privilege `permissions:` at the workflow top level (e.g. `permissions: contents: read`) and elevate only in specific required jobs.
- Use intermediate environment variables to safely handle untrusted context values (`github.event.issue.title`, `github.head_ref`) in inline shell scripts to prevent script injection.
- Mask secrets and avoid exposing sensitive tokens in step outputs, logs, or artifact uploads.
- Restrict `pull_request_target` workflows to purely read-only operations or ensure PR code is never checked out and executed with write tokens.

## Blocking anti-patterns

- Using unpinned or mutable branch/tag references (`@main`, `@master`, `@v1`) for third-party actions.
- Defining broad `permissions: write-all` on workflows.
- Inlining user-controlled context variables directly into `run:` scripts (e.g. `run: echo "${{ github.event.comment.body }}"`).
- Printing secrets or base64-encoded credentials directly into GitHub Actions job logs.
- Triggering untrusted code execution in workflows with access to deployment secrets.

## Minimum checks

- Run `actionlint` or workflow schema validation across `.github/workflows/*.yml`.
- Verify workflow syntax using dry-run tools (e.g. `act` or GitHub CLI workflow validations).
- Ensure required status check names match branch protection rules exactly.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
