# POSE — Project Operating Standard for Engineering

**Turn AI-assisted engineering into a repository-owned, machine-checkable
delivery system.**

POSE is the free, Apache-2.0 governance core for teams building software with
humans and AI agents. It installs an operating contract in the repository and
enforces that contract with one native Go binary:

```text
spec → execution → evidence → follow-ups → recurrence → knowledge
  ▲                                                        │
  └────────────── learning returns to planning ─────────────┘
```

POSE is not another coding agent, IDE or project board. It is the layer that
makes work portable across those tools: what may start, which rules apply,
which checks must pass, what evidence proves completion and what the next
execution needs to remember.

## Why POSE

AI coding tools accelerate implementation, but speed alone does not solve the
system-level problems they amplify:

- requirements remain trapped in chat history;
- agents receive inconsistent instructions;
- “done” is declared without reproducible evidence;
- follow-ups disappear into prose;
- the same failures are fixed repeatedly;
- context is lost when an agent or session changes.

POSE makes each of those concerns an explicit, versioned mechanism.

| Differentiator | What POSE does | Verifiable mechanism |
|---|---|---|
| **Governs delivery, not only generation** | Connects planning, execution, acceptance and learning | Specs + workflows + rules + evidence + history |
| **Gates both entry and exit** | Refuses execution without readiness and refuses done without closeout | `pose lint-spec --ready-check` / `--strict` |
| **Uses real engineering checks** | Runs repository-native test, lint, typecheck and build commands | `validation-matrix.json` + `pose validate` |
| **Turns evidence into memory** | Stores versionable reports and append-only history | `.pose/reports/` + `pose report` |
| **Closes residual work** | Requires a disposition for every follow-up | `pose followups` + closeout vocabulary |
| **Escalates systemic failure** | Detects recurring task failures and routes structural correction | `pose recurrence-check` + escalation workflow |
| **Preserves operational context** | Gives handoffs and decisions an owner, sensitivity and TTL | `.pose/knowledge/` + `knowledge-check` |
| **Plans from dependencies** | Validates spec and milestone DAGs and computes readiness | `depends_on`, roadmaps, `pose_spec_readiness` |
| **Works across agents** | Exposes short instructions, portable skills and MCP tools | `AGENTS.md`, Agent Skills, `pose serve-mcp` |
| **Keeps control local** | Runs offline and stores the source of truth in Git | One CGO-free binary; no hosted dependency |

## Where POSE is strongest

POSE's core advantage is **closed-loop governance**. Spec authoring is only the
first step. The system also checks whether a spec is ready, routes the correct
workflow, executes deterministic quality gates, records the acceptance
evidence, forces residual work to be triaged and detects when local fixes should
become systemic improvements.

That combination is especially valuable for:

- teams using more than one coding agent or model provider;
- brownfield repositories where architecture and checks already exist;
- regulated or high-accountability delivery;
- monorepos with different stacks and module criticalities;
- platform teams standardizing engineering without forcing one IDE;
- organizations preparing for governed agent orchestration.

If you only need a prompt template or a lightweight planning folder, POSE may
be more structure than you need. Start with Spec Kit or OpenSpec and import the
result later; POSE includes native, safe importers for both.

## How POSE compares

These products solve adjacent problems and can be complementary. The useful
question is not “which tool wins?” but “which part of delivery does each tool
make authoritative?”

| Solution | Primary strength | POSE's distinction |
|---|---|---|
| [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md) | Rich SDD lifecycle with agent integrations, extensions, presets, workflows and bundles | POSE emphasizes repository-wide governance after planning: entry/exit gates, validation evidence, recurrence and expiring knowledge |
| [OpenSpec](https://github.com/Fission-AI/OpenSpec) | Lightweight, agent-neutral brownfield change proposals, deltas and archive-to-source flow | POSE adds deterministic delivery gates, module-aware validation, governed follow-ups, operational history and portfolio readiness |
| [Kiro](https://aws.amazon.com/documentation-overview/kiro/) | Integrated agentic service with specs, steering and event hooks | POSE is editor/model neutral, offline and owned by the repository |
| [Backstage](https://backstage.io/docs/features/software-catalog/) | Organization-wide software catalog, templates and developer portal | POSE governs execution inside each repository and can feed a portal/control plane |
| CI orchestrators | Execute pipelines and display job results | POSE decides the applicable trail, normalizes severity and preserves evidence as governed product data |
| Issue trackers | Coordinate people, status and portfolio work | POSE makes the engineering contract and acceptance criteria executable beside the code |

POSE does not replace the specialist strengths above. It provides the
governance spine that remains stable while agents, editors, CI providers and
portals change.

## The free core and the scale path

POSE is the open-source entry point to the broader **Harne8** platform.

| Start with POSE | Scale with Harne8 |
|---|---|
| Repository-local specs and roadmaps | Visual multi-project portfolio |
| Workflows, rules and portable skills | Durable task orchestration through Conductor |
| Deterministic validation and evidence | Governed agent execution through Harness |
| Local insights and recurrence | Central reliability, cost and policy views |
| Native MCP governance API | Context enrichment through GraphForge |
| Optional OPA policy enforcement | Central identity, approvals, audit and operations |

The boundary is intentional: the free core remains useful by itself, offline
and vendor neutral. Harne8 adds coordination and visual operation when
repository-local governance is no longer enough.

## What is in the box

| Path or component | Purpose |
|---|---|
| `pose` binary | Native CLI, installer, gates, reports, metrics, housekeeping and MCP |
| `.pose/specs/` | Living feature contracts with lifecycle and dependencies |
| `.pose/workflows/` | Procedures for feature, bugfix, review, refactor, docs and recurrence |
| `.pose/rules/` | Cumulative security, backend, frontend, Kubernetes, evidence and knowledge rules |
| `.agents/skills/` | Nine portable Agent Skills; Claude-compatible links are installed |
| `.pose/roadmaps/` | Governed roadmaps with milestone DAGs and readiness |
| `.pose/knowledge/` | TTL-governed handoffs, notes and decision logs |
| `.pose/reports/` | Versionable evidence and append-only JSONL history |
| `.pose/indexes/` | Repository, module, task, spec-graph and roadmap projections |
| `pose serve-mcp` | 20 POSE tools over stdio or Streamable HTTP |
| `mcp-enforce/` | Optional project/run-scoped identity, OPA decisions and audit |
| `pose-action/` | GitHub Action adapter for deterministic gates |

Read the [technical architecture](docs-site/docs/architecture.md) for every
component and mechanism. Read the
[capability assessment](docs-site/docs/capability-assessment.md) for current
maturity and best-of-breed gaps. The governed
[product roadmaps](docs-site/docs/product-roadmaps.md) convert those findings
into 7 roadmaps, 35 implementation specs and dependency-aware release gates.

## Quickstart

Download the released archive for your platform, verify its checksum, place
`pose` on `PATH`, then install POSE into a Git repository. Release assets are
named `pose_<version>_<os>_<arch>` — `tar.gz` for Linux and macOS, `zip` for
Windows — on `linux`/`darwin`/`windows` × `amd64`/`arm64`.

Linux and macOS (bash or zsh; replace `linux_amd64` with your platform):

```bash
V=0.9.0
curl -fsSLO "https://github.com/oseiaspereira88/pose/releases/download/v${V}/pose_${V}_linux_amd64.tar.gz"
curl -fsSLO "https://github.com/oseiaspereira88/pose/releases/download/v${V}/checksums.txt"
sha256sum --check --ignore-missing checksums.txt   # macOS: shasum -a 256 -c
tar -xzf "pose_${V}_linux_amd64.tar.gz" pose
install -m 0755 pose ~/.local/bin/pose             # any directory on PATH
pose install /path/to/your/repo
```

Windows (PowerShell):

```powershell
$V = "0.9.0"
Invoke-WebRequest "https://github.com/oseiaspereira88/pose/releases/download/v$V/pose_${V}_windows_amd64.zip" -OutFile "pose_${V}_windows_amd64.zip"
Invoke-WebRequest "https://github.com/oseiaspereira88/pose/releases/download/v$V/checksums.txt" -OutFile checksums.txt
(Get-FileHash "pose_${V}_windows_amd64.zip" -Algorithm SHA256).Hash -eq ((Get-Content checksums.txt | Select-String "pose_${V}_windows_amd64.zip") -split '\s+')[0]
Expand-Archive "pose_${V}_windows_amd64.zip" -DestinationPath .
# move pose.exe to a directory on PATH, then:
pose install C:\path\to\your\repo
```

Always verify the checksum before executing the binary. Never pipe downloaded
scripts into a shell: the optional `install.sh` in the release bundle is meant
to be downloaded next to the verified binary and run locally.

The installer:

- embeds workflows, rules, templates, skills and the selected locale;
- derives project name and ID, with explicit override flags;
- configures the same binary as the MCP server;
- preserves existing specs, ADRs, knowledge, reports and roadmaps;
- finishes with native `init`, `index` and `check --strict`;
- reports success only when the structural gate passes.

Requirements: Git and the native `pose` binary. Bash is needed only when using
the optional release-bundle `install.sh`; the runtime itself needs no Bash,
Python, Node.js or hosted service. Supported release targets are Linux, macOS
and Windows on `amd64` and `arm64`.

Every release publishes `compatibility.json` (supported engine, schema and
upgrade pairs) and the generated `compatibility-report.md` (the release gate
evidence) as release assets. Binary SemVer and repository schema compatibility
are independent axes: `pose upgrade` migrates an instance forward through
ordered idempotent migrations; downgrade is unsupported by contract.

## Run a first governed delivery

```bash
pose init --wizard --yes
pose new-spec customer-export
pose suggest feature

# Fill Intent, R1/R2... requirements and Technical Plan.
pose lint-spec customer-export --ready-check

# Implement, then run the repository's declared checks.
pose validate --strict
pose report --task "customer-export" --spec customer-export

# Stamp completed_at and disposition every follow-up before done.
pose lint-spec customer-export --strict
```

Already use another SDD format?

```bash
pose import spec-kit .specify/specs --dry-run
pose import openspec openspec/changes/add-2fa --dry-run
```

The importer validates the complete batch before writing, rejects symlinks,
never overwrites an existing spec and reports everything that still needs
human curation.

## Adopt progressively

1. **Observe:** install POSE and run checks in tolerant mode.
2. **Align:** customize module metadata, rules and the validation matrix.
3. **Enforce:** make stable required checks blocking in CI.
4. **Learn:** generate reports, triage follow-ups and enable recurrence checks.
5. **Scale:** connect MCP clients or Harne8 without moving the source of truth
   out of the repository.

Teams using pre-commit.com can enable `pose-check`, `pose-lint-spec` and
`pose-history-check`. See the [CI guide](docs-site/docs/ci.md) and
[CLI reference](docs-site/docs/cli.md).

## Security and privacy

- Gates are offline by contract.
- Telemetry is disabled by default and has no built-in collection endpoint.
- Import and module paths are confined to the project root.
- Validation uses structured program/argument arrays; legacy shell commands are
  rejected.
- Restricted knowledge is excluded from MCP reads.
- OPA-backed MCP policy fails closed on evaluation errors.
- Mutating repository work remains an execution-sandbox responsibility rather
  than a general MCP write surface.

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## Current product boundary

POSE currently provides a strong local governance engine, not a hosted
multi-team service. Its local reports are auditable Git artifacts, not signed
supply-chain attestations. Roadmaps express dependency/readiness, not team
capacity. Local insights summarize POSE outcomes, not deployment or incident
performance.

Those limits are explicit in the
[capability assessment](docs-site/docs/capability-assessment.md), alongside the
work required to reach the next maturity level.

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).

Contributions are welcome: see [CONTRIBUTING.md](CONTRIBUTING.md).
POSE is developed as the governance plane of the **Harne8** AI-native
engineering platform.
