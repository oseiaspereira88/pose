# CI integration

## GitHub Action

The distribution ships a composite action (`pose-action/`):

```yaml
jobs:
  pose:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: <owner>/<repo>/pose-action@main   # marketplace name after the repo split
        with:
          mode: strict            # or tolerant
          lint-specs: "true"
          recurrence-check: "true"
          history-check: "true"
```

It runs `pose check`, `pose lint-spec --all`, `pose recurrence-check` and
`pose history-check` — all offline, needing only bash + python3 (present on
all GitHub runners).

## Use POSE from pre-commit.com

Require pre-commit 4.4 or newer and install POSE in the repository first. Pin
the POSE repository to an immutable release tag or commit:

```yaml
repos:
  - repo: https://github.com/oseiaspereira88/pose
    rev: v0.2.0  # replace with the first release containing these hooks
    hooks: [{id: pose-check}, {id: pose-lint-spec}, {id: pose-history-check}]
```

Run `pre-commit install`, then use `pre-commit run --all-files` in CI. The
hooks call the repository's installed `./pose` in strict mode and do not
receive staged filenames. Run one manually with
`pre-commit run pose-check --hook-stage manual --all-files`. Skip a single
hook temporarily with `SKIP=pose-history-check git commit ...`; CI remains the
delivery authority and should not skip required gates.

## Recommended rollout

1. **Observability first**: run the action in `tolerant` mode on PRs; publish
   logs as artifacts; raise no new gates.
2. **Enforce on main**: switch to `strict`; adjust `moduleOverrides` for
   modules that aren't ready instead of weakening the default.
3. **Promote checks**: move stable `optional` checks to `required` per domain.
4. **Harden**: review the matrix periodically; remove temporary exceptions.

## Releases

Tagging `v*` triggers the release pipeline: POSE gates + Go tests + installer
E2E, then goreleaser publishes multi-platform binaries (`pose`, `pose-mcp`)
with SHA-256 checksums, the installer script, and release notes consolidated
from the POSE changelog fragments. See `docs/RELEASE.md` in the repository for
the full process and ownership of the remaining manual steps.
