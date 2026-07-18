# POSE

**Spec-driven development that closes the loop.**

POSE (Project Operating Standard for Engineering) is a governance framework
for agentic software engineering. Where most SDD tools stop at scaffolding a
spec, POSE governs the full cycle — and enforces it with deterministic gates:

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
| **Native MCP surface** | 18 read tools expose the whole instance (specs, readiness, roadmaps, knowledge, reports, insights, changelogs) to any MCP-capable agent. |

## How POSE compares

GitHub's spec-kit, OpenSpec and similar SDD tools generate well-structured
specs and prompts — they govern the *entry* of work. POSE governs entry **and
exit**: the Definition-of-Ready gate is matched by a closeout gate that
refuses "done" until evidence is recorded and every follow-up is triaged, and
the history those gates produce feeds recurrence detection and
portfolio-level readiness. If you only need spec templates, lighter tools are
fine; POSE is for teams that want the loop closed and machine-checkable.

## License

Apache-2.0. POSE is developed as part of the **Crisol** platform and runs
entirely offline in your repository — the governance layer is free, forever.
