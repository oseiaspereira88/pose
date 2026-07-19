# ADR: Validation runtime guardrails and Harness delegation

## Status
Accepted (2026-07-19) — spec `pose-validation-runtime-guardrails`

## Context

`pose validate` executed matrix checks with no time or output bounds: a hung
test could block CI forever and a flooding (or hostile) check could exhaust
the runner. The roadmap's risk control is explicit — keep sandbox execution
in the Harness rather than weakening the local CLI boundary — and the spec's
non-goal forbids turning the CLI into a remote execution service.

Alternatives considered:

1. **No local limits, rely on CI job timeouts** — coarse (kills the whole
   run, not the offending check) and invisible in the result contract.
2. **Local sandboxing (namespaces/containers)** — platform-specific, heavy,
   and exactly the boundary-weakening the roadmap forbids.
3. **Portable local guardrails + explicit Harness delegation contract.**

## Decision

Option 3:

- **Timeout (R1):** per check `timeoutSeconds`, falling back to
  `defaults.timeoutSeconds`, then the documented safe default (600s).
  Cancellation kills the child's whole process group on Unix
  (`Setpgid` + group SIGKILL via `Cmd.Cancel`); on platforms without
  process groups the direct child is killed and the limitation is
  documented here. The result records the explicit state
  (`outcome: error`, `limit_state: timeout`) with elapsed time.
- **Output ceiling (R1):** `defaults.maxOutputBytes` (default 1 MiB);
  exceeding it cancels the check and records
  `limit_state: output-limit`. This is distinct from the 4 KiB capture
  tail — the ceiling is a kill switch, the tail is evidence hygiene.
- **Isolation classification (R2):** matrix checks may declare
  `isolation: "required"`. The local CLI never executes them — they are
  recorded as skipped with a machine-readable reason. Existing checks
  default to local execution with the safe limits above (compatibility).
- **Harness delegation (R3):** `--emit-plan <file>` writes an execution-plan
  envelope binding project id, spec/task, git HEAD, the SHA-256 of the
  exact validation matrix, the isolation-required check plan and an
  approval slot (`approval.required: true`, identity/expiry empty). The
  plan is inert until the control plane stamps an expiring execution
  identity (ADR-007 model); the CLI authors contracts, never remote
  execution.
- **Explicit states over silent degradation:** timeouts and ceilings are
  guardrail `error` states — never conflated with check failures (`fail`)
  and never silently tolerated: severity still decides blocking.

## Consequences

- Positive: a hung or flooding check can no longer freeze validation, and
  the result explains exactly what was bounded and why.
- Positive: hostile-check execution has a governed path (plan → approval →
  Harness) instead of a local workaround.
- Trade-off: default limits can interrupt legitimately slow checks; both
  are configurable per matrix and the interruption is visible, not silent.
- Residual: on non-Unix platforms descendants of a killed child may
  survive; documented, and the Harness path is the answer for untrusted
  workloads.
