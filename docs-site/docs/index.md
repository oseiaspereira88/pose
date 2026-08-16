# POSE

**Doc type:** Explanation &nbsp;·&nbsp; **Applies to:** POSE 1.4.x

**Repository-owned governance for agentic engineering.**

POSE (Project Operating Standard for Engineering) is the local operating
contract around humans, coding agents and CI. It turns intent, policy,
execution, evidence and learning into versioned artifacts and deterministic
gates:

```text
discover → specify → route → execute → prove → close → learn
    ▲                                              │
    └──────────── repository-owned context ────────┘
```

It is not another coding agent and does not replace your language toolchain.
Agents propose and execute changes; POSE defines the conditions under which
those changes are ready, valid, reachable and complete.

## What makes POSE different

| Mechanism | What it changes |
|---|---|
| **Executable repository contract** | `AGENTS.md`, workflows, cumulative rules and task-aware skills travel with the code instead of living only in prompts or a hosted board. |
| **Entry and exit gates** | Definition of Ready blocks under-specified work; closeout requires completion metadata, requirement evidence and explicit follow-up disposition. |
| **Deterministic proof** | Module-aware test/lint/typecheck/build checks emit one canonical result as text, JSON, JUnit or SARIF. |
| **Convergent technical review** | Component-aware plans select the relevant criteria and tools; a sealed semantic/source bundle is approved by a separate append-only attestation. |
| **Composition assurance** | Artifact and surface gates distinguish “a file exists” from “the capability is reachable through a production entrypoint with current evidence.” |
| **Operational learning** | Follow-ups, recurrence effectiveness and expiring knowledge keep residual work and reusable context alive without turning every note into permanent policy. |
| **Verifiable distribution** | The native binary ships with checksums, keyless Sigstore signatures, CycloneDX SBOMs, SLSA provenance and independent rebuild verification. |

## One engine, three interfaces

- **CLI:** scaffold, assess, validate, report, close and maintain from one native
  Go binary, offline by design.
- **MCP:** 47 POSE governance tools plus 3 optional Conductor run reporters.
  The catalog is versioned and frozen against a golden fixture; project scope,
  policy and safe validation orchestration are explicit.
- **CI:** GitHub Action, pre-commit hooks and the same validation matrix promote
  local policy into reproducible delivery gates.

## Analytics without manual counters

POSE automatically observes recognized local CLI commands and authorized
project-backed MCP calls. `pose usage` reports tools, outcomes, latency and the
lifecycle of structured findings without asking agents to maintain `count++`.
The privacy-bounded journal stays outside the worktree and excludes arguments,
output, paths, source content and personal/project identity.

Adoption metrics derive first value and retention from POSE's own governed
history. DORA uses explicit deployment and incident events to report the five
current, production-scoped metrics. These planes remain intentionally separate:
tool activity never masquerades as delivery performance. See
[Analytics and delivery metrics](analytics.md).

## Start here

Install the latest Linux/macOS release from the root of your Git repository:

```bash
curl -fsSLO https://github.com/oseiaspereira88/pose/releases/latest/download/install.sh && bash install.sh
```

For a pinned, checksum- and Sigstore-verified installation, use the
[Quickstart](quickstart.md#verified-install).

1. Follow the [Quickstart](quickstart.md) to install v1 and close a first loop.
2. Read [Concepts](concepts.md) for lifecycle, evidence and knowledge semantics.
3. Use the [CLI reference](cli.md) or [MCP reference](mcp.md) for exact surfaces.
4. Review the [technical architecture](architecture.md) and the evidence-based
   [capability assessment](capability-assessment.md) before extending policy.

Existing GitHub Spec Kit and OpenSpec work can be imported natively and
reviewed before it enters the POSE lifecycle. Extensions add skills, workflows,
rules and import adapters without requiring a fork.

## License and boundary

POSE core is Apache-2.0 and runs entirely in your repository without a Harne8
account or mandatory telemetry. The optional Harne8 Platform composes durable
multi-team orchestration, visual operation and centralized policy distribution;
it is not an artificial gate inside the local engine.
