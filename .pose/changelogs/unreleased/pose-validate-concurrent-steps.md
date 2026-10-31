---
spec: pose-validate-concurrent-steps
category: added
breaking: false
---
`pose validate` runs independent checks concurrently: `--jobs <n>` (default from cores and available memory, at most 8), results in matrix order whatever order they finish in, `exclusive: true` for a check that must run alone, and the run's wall time, sum of checks and slowest checks at the end. An interrupt or SIGTERM now stops every running check's process group and starts nothing more. A strict validation of pose-dist took about half the wall time at `--jobs 4` (400–447 s against 825 s, same machine and settings).
