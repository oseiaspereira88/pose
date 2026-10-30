# Launcher parity check — first run, failing

Spec `pose-delegation-parity-gate` R3: the parity check was run against the first complete draft of `.pose/parity/delegation-launchers.json` on 2026-10-29, at `a456b8f73cc1086c54f6f1af7e253ea5a9ac01f3` with the matrix and the check uncommitted, before any gap it found was closed. It failed; the output below is kept verbatim as evidence that the check fails before it passes.

Command: `go test ./internal/cli -run LauncherParity -count=1 -v` in `pose-mcp`.

```text
=== RUN   TestLauncherParityMatrixCoversEveryLauncherCapability
    launcher_parity_test.go:177: the launchers are not at parity:
          SA-02: no covering test
          SA-07: command names --prompt, which `pose delegate resume` does not document
          SA-08: command names --wait-until-reset, which `pose delegate resume` does not document
          SA-12: command names --json, which `pose delegate task` does not document
          SW-05: no covering test
--- FAIL: TestLauncherParityMatrixCoversEveryLauncherCapability (0.04s)
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow/no_covering_test
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow/a_test_that_does_not_exist
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow/no_native_command
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow/a_command_that_does_not_exist
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow/a_flag_the_command_does_not_document
=== RUN   TestLauncherParityCheckFailsOnEveryUncoveredRow/an_ADR_capability_with_no_row
--- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow (0.04s)
    --- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow/no_covering_test (0.00s)
    --- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow/a_test_that_does_not_exist (0.00s)
    --- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow/no_native_command (0.00s)
    --- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow/a_command_that_does_not_exist (0.00s)
    --- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow/a_flag_the_command_does_not_document (0.00s)
    --- PASS: TestLauncherParityCheckFailsOnEveryUncoveredRow/an_ADR_capability_with_no_row (0.00s)
FAIL
FAIL	github.com/harne8/pose-mcp/internal/cli	0.085s
FAIL
```

What each line found: SA-02 and SW-05 had no covering test (the same-vendor warning and `watch -v --lines`); `pose delegate resume` did not document `--prompt`, `pose delegate task` did not document `--json`, and `--wait-until-reset[=<max>]` was not recognized by the check's flag reader. The fixture test passed: each kind of uncovered row is reported.
