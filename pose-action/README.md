# POSE Gates — GitHub Action

Runs the deterministic [POSE](../README.md) governance gates on any
repository with an installed POSE instance.

```yaml
jobs:
  pose:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # path form; pin to a release tag or commit SHA in production
      - uses: oseiaspereira88/pose/pose-action@main
        with:
          mode: strict            # or tolerant
          lint-specs: "true"
          recurrence-check: "true"
          history-check: "true"
```

Requirements in the target repo: an installed POSE instance (`pose`,
`.pose/`) — see the [installer](../README.md#quickstart). The gates are offline
and require the native `pose` binary on `PATH` plus git.
