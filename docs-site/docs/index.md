# POSE

**Doc type:** Explanation &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

**Spec-driven development that closes the loop.**

POSE (Project Operating Standard for Engineering) is a repository-owned
governance framework for agentic software engineering. It complements SDD
authoring tools by governing the full delivery cycle with deterministic gates:

```
spec → execution → evidence → follow-ups → recurrence → knowledge
  ▲                                                        │
  └────────────── the loop closes back into planning ──────┘
```

## Why POSE

| Capability | What it means |
|---|---|
| **Definition of Ready gate** | A spec cannot enter execution without intent, requirements with stable acceptance-criteria IDs and a technical plan (`pose lint-spec --ready-check`). |
| **Closeout gate** | A spec cannot be marked done until its completion date is stamped and *every* follow-up is explicitly dispositioned (`pose lint-spec --strict`). |
| **Deterministic validation** | A per-stack/per-module matrix runs real commands (`test`, `lint`, `typecheck`, `build`) and produces versionable reports with append-only JSONL history. |
| **Recurrence detection** | `pose recurrence-check` mines that history for repeated failures and escalates them into systemic fixes instead of endless re-fixing. |
| **Operational memory** | `.pose/knowledge/` captures handoffs and decision logs with TTL governance — context survives between executions and agents. |
| **Portfolio graph** | Specs declare dependencies and priorities, organized into governed roadmaps with milestone DAGs, validated on every `pose check`. |
| **Native MCP surface** | 20 governance tools (read + deterministic gates) expose specs, readiness, requirement traces, amendment history, roadmaps, knowledge, reports, insights and changelogs to MCP-capable agents, plus 3 optional Conductor run reporters. |

## How POSE compares

[GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)
provides a rich SDD lifecycle and extension model; [OpenSpec](https://github.com/Fission-AI/OpenSpec)
focuses on lightweight, agent-neutral change proposals and spec deltas;
[Kiro](https://aws.amazon.com/documentation-overview/kiro/) integrates specs,
steering and hooks into an agentic development service.
POSE's focus is the repository-wide operating contract around those planning
artifacts: entry and exit gates, module-aware validation, versioned evidence,
follow-up disposition, recurrence and expiring operational knowledge.

Read the [technical architecture](architecture.md) for the complete component
and mechanism model. Use the [capability assessment](capability-assessment.md)
for evidence-based maturity and benchmark gaps, then use the governed
[product roadmaps](product-roadmaps.md) for implementation order, milestones
and release gates.

## License

Apache-2.0. POSE is developed as part of the **Harne8** platform and runs
entirely offline in your repository — the governance layer is free, forever.
