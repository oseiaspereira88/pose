# CI integration

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE 1.2.x

## GitHub Action

The distribution ships a composite action (`pose-action/`):

```yaml
jobs:
  pose:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oseiaspereira88/pose/pose-action@main   # pin to a release tag or commit SHA in production
        with:
          mode: strict            # or tolerant
          lint-specs: "true"
          recurrence-check: "true"
          history-check: "true"
```

It runs `pose check`, `pose lint-spec --all`, `pose recurrence-check` and
`pose history-check` — all offline, requiring the native `pose` binary and git.

## Use POSE from pre-commit.com

Require pre-commit 4.4 or newer and install POSE in the repository first. Pin
the POSE repository to an immutable release tag or commit:

```yaml
repos:
  - repo: https://github.com/oseiaspereira88/pose
    rev: v1.4.0  # pin to an immutable release tag
    hooks: [{id: pose-check}, {id: pose-lint-spec}, {id: pose-history-check}]
```

Run `pre-commit install`, then use `pre-commit run --all-files` in CI. The
hooks call `pose` from `PATH` in strict mode and do not
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

`pose release plan` previews a version, and `pose release prepare --apply`
freezes the exact changelog fragments, canonical notes and candidate manifest.
Tagging that prepared candidate triggers POSE gates, Go tests, installer E2E
and GoReleaser. Publication emits the multi-platform binary, SHA-256 checksums,
Sigstore bundles, per-archive CycloneDX SBOMs and SLSA provenance; the release
lifecycle then records tagged/published/verified evidence without rewriting
the tag or mutable notes. See the
[release lifecycle](cli.md#release-lifecycle) for the governed command flow.
