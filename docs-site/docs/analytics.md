# Analytics and delivery metrics

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE 1.0.x

POSE exposes three measurement planes. They answer different questions and
must remain separate:

| Plane | Question | Source | Command |
|---|---|---|---|
| **Usage** | Which POSE tools are used, and what do their gates find? | Automatic local CLI/MCP execution events | `pose usage` / `pose_usage` |
| **Adoption** | Is the team reaching and repeating the POSE workflow? | Specs and governed history already owned by POSE | `pose adoption-metrics` |
| **Delivery (DORA)** | How frequently and safely does an application reach an environment? | Explicit deployment and incident events | `pose dora-metrics` |

Usage is not delivery performance. A frequently used gate can reveal many
problems because it is valuable, because the codebase is unhealthy, or both.
POSE reports the observation and does not invent causation.

## Understand tool usage

No agent maintains counters. The native CLI wrapper records recognized
commands, and the MCP dispatcher records authorized project-backed tool calls
when they reach a terminal result. Introspection and long-running control
commands such as `usage`, `help`, `version`, `telemetry` and `serve-mcp` are
excluded.

```bash
# All observed POSE activity in the default 30-day window.
pose usage

# Compare one surface or one tool.
pose usage --since-days 90 --surface mcp
pose usage --tool pose_check --json
```

The report separates:

- **transport/execution outcome** — whether the command or MCP call completed;
- **semantic outcome** — whether a deterministic gate passed, warned or failed;
- **finding observations** — every structured finding seen across executions;
- **unique/new findings** — stable findings, counted once when first observed;
- **resolved/reopened findings** — lifecycle transitions calculated only when
  complete finding sets from comparable tool/surface/scope executions exist;
- **latency** — bounded duration aggregates and sample metadata.

Structured validation contributes exact failed/errored check counts and stable
check identities. A generic command failure contributes one conservative
failed observation; POSE never parses arbitrary stderr lines into invented
findings.

### Interpret usefulness without vanity metrics

Use call volume to understand adoption, semantic pass/fail to understand gate
outcomes, and finding lifecycle to understand whether detected problems move.
A useful review asks:

1. Which tools are repeatedly reached by real workflows?
2. Which gates produce new findings rather than only repeating the same ones?
3. Are findings later resolved or reopened within a comparable scope?
4. Are high-latency tools valuable enough to keep in the critical path?

Do not rank people or agents by calls, failures or findings. These signals are
project-local product analytics, not individual productivity measures.

### Privacy and storage

Usage recording is offline, best-effort and outside the tracked worktree. POSE
stores an allowlisted event schema with tool/surface, bounded outcomes, counts,
duration, engine version and project-local HMAC fingerprints. It does **not**
store command arguments, output, paths, repository/project names, principals,
run ids, source content or raw finding ids. Nothing is transmitted.

The default location is resolved from the Git common directory; a non-Git
project falls back to an OS user-cache directory keyed by a root hash. Operators
may set `POSE_USAGE_DIR` to an absolute local directory, for example a
persistent container mount. Set `POSE_USAGE_DISABLED=1` to stop new events;
existing events remain queryable.

!!! note "Human confirmation is a recorded evolution"

    POSE 1.0.0 does not infer whether an observed finding is `valid`,
    `wont-fix` or `false-positive`. That explicit adjudication is an owned
    follow-up of `pose-usage-metrics`. Automatic observation counts remain
    separate so a future human decision cannot rewrite what the gate saw.

## Measure adoption

```bash
pose adoption-metrics --json
```

The report derives activation, time-to-first-gate, retention and task success
from specs and history POSE already owns. It needs no deployment event and no
remote telemetry. Use it to identify onboarding friction and whether governed
work repeats after first value; do not interpret it as production reliability.

## Record delivery events for DORA

DORA is explicit input only. POSE never guesses a production deployment from a
commit and never treats usage as a deployment.

```bash
pose record-deployment \
  --application checkout \
  --environment production \
  --deployment-kind planned \
  --status success \
  --source ci \
  --lead-time-seconds 5400

pose record-incident \
  --application checkout \
  --environment production \
  --started-at 2026-08-11T12:00:00Z \
  --resolved-at 2026-08-11T12:45:00Z \
  --severity major \
  --source webhook \
  --caused-by-deployment

pose dora-metrics \
  --application checkout \
  --environment production \
  --window-days 30 \
  --json
```

The five v1 metrics are:

1. deployment frequency;
2. lead time for changes;
3. change failure rate;
4. failed deployment recovery time;
5. deployment rework rate.

Every denominator is scoped to the selected environment (`production` by
default). Recovery includes only resolved incidents explicitly marked as
caused by a deployment. Rework depends on `deployment_kind=planned|rework`;
if a selected window contains legacy deployments without that classification,
only that metric is `unavailable` rather than a misleading zero.

## Keep telemetry separate

`pose telemetry` is optional, anonymous product telemetry with an inspectable
payload. It is unrelated to the local usage journal and DORA event store.

```bash
pose telemetry status
pose telemetry enable   # only after reviewing the payload and policy
pose telemetry disable
```

The default remains disabled. Enabling telemetry does not export the usage
journal, repository content or DORA events.

## Operational review cadence

A practical team cadence is:

- weekly: inspect `pose usage` for unused critical gates, repeated findings and
  surprising latency;
- monthly: inspect adoption for first-value and retention friction;
- per delivery review: inspect DORA by application and production environment;
- after policy or workflow changes: compare finding recurrence and task outcome
  history before claiming the intervention worked.

Retain and delete deployment/incident events with
`pose events-housekeeping`; the usage journal remains local machine state and
is deliberately not a Git-backed audit record.
