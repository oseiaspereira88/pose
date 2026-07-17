# POSE Gates — GitHub Action

Runs the deterministic [POSE](../pose-dist/README.md) governance gates on any
repository with an installed POSE instance.

```yaml
jobs:
  pose:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # pre-extraction path form; after the POSE repo split this becomes
      # uses: <owner>/pose-action@v1 (marketplace)
      - uses: <owner>/<repo>/pose-action@main
        with:
          mode: strict            # or tolerant
          lint-specs: "true"
          recurrence-check: "true"
          history-check: "true"
```

Requirements in the target repo: an installed POSE instance (`./pose`,
`.pose/`) — see the [installer](../pose-dist/README.md#quickstart). The gates
are offline and need only bash + python3 (present on all GitHub runners).
