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
