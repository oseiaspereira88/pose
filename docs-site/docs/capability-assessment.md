# Capability assessment

**Assessment date:** 2026-07-18  
**Evidence baseline:** commit `d9c0b98`, local source inspection, `pose doctor
--json`, MCP `tools/list` and repository checks  
**Purpose:** measure fitness against POSE's own promises and relevant
best-of-breed practices, not manufacture a universal product ranking.

## Method

Score each mechanism against its stated purpose:

| Score | Interpretation |
|---:|---|
| 0 | Absent |
| 1 | Designed or scaffolded |
| 2 | Functional in a narrow path |
| 3 | Reliable for a single repository or early team adoption |
| 4 | Strong product capability with clear operational contracts |
| 5 | Reference-grade capability with ecosystem, scale and independently verified evidence |

The target is not always 5. A local engine should not become a hosted portal
only to improve a score. Compare each mechanism with the strongest relevant
practice while preserving POSE's boundary.

## Executive result

POSE is strongest where the market is usually fragmented: lifecycle closure,
repository-owned governance, deterministic gates, follow-up disposition and
recurrence feedback. It is already credible as an open-source, single-repo
governance engine. The largest distance to a reference-grade product is not in
the core SDD model; it is in distribution trust, ecosystem reach, measurable
delivery outcomes, public product polish and scaled team operation.

| Mechanism | Current | Target | Summary |
|---|---:|---:|---|
| Install, upgrade and local-first runtime | 4 | 5 | Strong native distribution; package channels and upgrade trust remain |
| Spec lifecycle and closeout | 4 | 5 | Distinctive two-sided gates; traceability can go deeper |
| Task routing, workflows, rules and skills | 4 | 5 | Coherent and agent-portable; extension lifecycle is still manual |
| Dependency graph, readiness and roadmaps | 3 | 4 | Useful eligibility model; limited portfolio planning |
| Validation matrix and structural checks | 4 | 5 | Safe structured execution; broader outputs and isolation needed |
| Evidence, history and insights | 3 | 5 | Auditable local history; no signed provenance or outcome integrations |
| Follow-ups and recurrence | 4 | 5 | High-value closed-loop mechanism; semantic and trend layers are early |
| Knowledge governance | 3 | 4 | TTL, ownership and sensitivity work; retrieval and team review can improve |
| MCP and agent interoperability | 3 | 5 | Broad read surface; contract drift and protocol completeness remain |
| Policy, identity and audit | 3 | 5 | Sound OPA/default-deny foundation; deployment hardening remains external |
| CI, release and supply-chain trust | 3 | 5 | Working pipelines/checksums; signatures, SBOM and provenance are missing |
| Import and adoption interoperability | 4 | 5 | Safe Spec Kit/OpenSpec import; no bidirectional or plugin ecosystem |
| Metrics and observability | 2 | 4 | Local outcome counts exist; DORA, traces and product analytics do not |
| Documentation, localization and diagnostics | 4 | 5 | Solid English/pt-BR and doctor; public onboarding still has placeholders |
| Extensibility and ecosystem | 2 | 4 | File contracts are extensible; no formal plugin/catalog lifecycle |
| Multi-repository and enterprise operation | 2 | 4 | MCP roots/policy exist; central UX, tenancy and durable orchestration live in Crisol |

## Detailed findings

### 1. Install, upgrade and local-first runtime — 4/5

**Purpose:** make governance easy to adopt without creating a hosted-service
dependency.

**Delivered now:** one CGO-free Go binary, six OS/architecture release targets,
embedded English and pt-BR scaffolds, idempotent installation, schema versioning,
dry-run upgrades and `pose doctor`.

**Strength:** a repository can keep operating offline with Git and its native
toolchains. Reinstallation preserves instance content.

**Gap to ideal:** publish supported Homebrew/Scoop/Winget/Nix or package-manager
paths, verify in-place version upgrades across released versions, align all
reported versions and publish a stable install command that does not assume the
binary is already present.

### 2. Spec lifecycle and closeout — 4/5

**Purpose:** prevent ambiguous work from starting and incomplete work from being
declared done.

**Delivered now:** stable requirement IDs, DoR checks, lifecycle dates,
dependency syntax, strict closeout and mandatory follow-up dispositions.

**Strength:** POSE couples an entry gate to an exit gate. This is a stronger
delivery-governance proposition than planning artifacts alone.

**Gap to ideal:** add requirement-to-test/result traceability, structured change
history for spec amendments and machine-readable approval records. Preserve
human review for intent changes.

**Benchmark:** [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)
offers a rich SDD lifecycle, extensions, presets and workflows;
[OpenSpec](https://github.com/Fission-AI/OpenSpec) is strong at lightweight
brownfield change deltas and archive-to-source flow; [Kiro](https://aws.amazon.com/documentation-overview/kiro/)
integrates specs, steering and hooks into an agentic development service.

### 3. Task routing, workflows, rules and skills — 4/5

**Purpose:** translate task intent into the correct procedure, constraints and
checks across agents.

**Delivered now:** ten routed task types, six workflows, cumulative domain
rules and nine portable skills, with Codex-native structure and Claude links.

**Strength:** routing is data, not prompt folklore. Rules can evolve without
copying every workflow.

**Gap to ideal:** validate the complete [Agent Skills](https://agentskills.io/specification)
contract in CI, version skills independently, add compatibility metadata and
publish a signed catalog with update/conflict handling comparable to mature
extension systems.

### 4. Dependencies, readiness and roadmaps — 3/5

**Purpose:** start only eligible work and expose planning order.

**Delivered now:** typed dependencies, priorities, acyclic spec/roadmap graphs,
milestone DAGs, target dates and MCP readiness resolution.

**Strength:** readiness derives from versioned contracts rather than board
status alone.

**Gap to ideal:** add impact visualization, critical-path explanation,
cross-repository references, ownership/capacity views and drift reconciliation.
Do not turn POSE into a transactional scheduler; expose these projections to
Crisol or portals such as [Backstage](https://backstage.io/docs/features/software-catalog/).

### 5. Validation and structural integrity — 4/5

**Purpose:** replace subjective “looks done” decisions with repeatable checks.

**Delivered now:** strict/tolerant modes, required/optional severity, structured
program/args/env execution, file predicates, module overrides and baseline
Node.js, Go, Rust and Java stacks.

**Strength:** POSE delegates to real test/build tools while rejecting shell-text
checks and confining module paths.

**Gap to ideal:** add Python, .NET and common monorepo/build systems; emit JUnit,
SARIF or a stable result schema; support timeouts and resource ceilings per
check; isolate untrusted repository checks in the Harness; add changed-scope
selection and cache adapters without making results nondeterministic.

### 6. Evidence, history and insights — 3/5

**Purpose:** preserve why a delivery was accepted and feed future decisions.

**Delivered now:** Markdown reports, append-only JSONL history, stable-field
hashes, sequences, changed-file capture, outcome aggregation and MCP insights.

**Strength:** evidence is inspectable with ordinary Git and survives ephemeral
CI logs.

**Gap to ideal:** link requirement IDs to checks and commits; capture actor and
approval identity; export standard event/result schemas; generate attestations.
POSE report hashes detect metadata change but are not signatures or build
provenance.

**Benchmark:** [SLSA](https://slsa.dev/spec/v1.0/levels) defines progressive
build provenance guarantees. CycloneDX defines interoperable SBOM media types
and predicates. These complement rather than replace POSE evidence.

### 7. Follow-ups and recurrence — 4/5

**Purpose:** stop residual work and repeated failures from disappearing into
free text.

**Delivered now:** disposition vocabulary, target validation, aggregated open
backlog, lexical duplicate candidates, time-window recurrence checks and an
escalation workflow.

**Strength:** findings become planning input, and repeated fixes are directed
toward systemic rules or workflows. This is POSE's clearest differentiator.

**Gap to ideal:** add owner/SLA for open follow-ups, trend and recurrence-cost
views, semantic candidate adapters with human confirmation, and effectiveness
measurement after an escalation is introduced.

### 8. Operational knowledge — 3/5

**Purpose:** preserve reusable context without creating permanent, ownerless
documentation.

**Delivered now:** handoff, decision-log and note types; owner, sensitivity,
review dates, TTL, overdue gate and explicit archive/purge operations.

**Strength:** knowledge has decay and accountability by design.

**Gap to ideal:** add reference validation from active work, review queues,
usage signals, optional semantic retrieval and external identity/RBAC mapping.
Keep restricted content excluded from broad MCP reads.

### 9. MCP and agent interoperability — 3/5

**Purpose:** expose governance consistently to any MCP-capable agent.

**Delivered now:** stdio and Streamable HTTP, 18 POSE tools, structured content,
multi-project roots and optional Conductor run reporting.

**Strength:** agents can query the same repository contracts the CLI enforces.

**Gap to ideal:** resolve the known catalog drift (`pose_validate` in ADR vs no
tool), advertise `project_id` consistently, derive MCP version from the binary,
add exact catalog conformance tests, pagination, resources/prompts where useful
and a defined refresh/reconnect contract. Publish and validate the registry
entry against each release.

**Benchmark:** the [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
defines schema-based discovery, pagination and invocation and recommends
human control over tool use.

### 10. Policy, identity and audit — 3/5

**Purpose:** authorize every remote governance call within project and run
scope.

**Delivered now:** bearer auth, OPA decisions, default denial on failure,
principal/project extraction, expiring HMAC execution identities and allow/deny
audit events.

**Strength:** policy decisions are separated from enforcement in the model
recommended by [OPA](https://www.openpolicyagent.org/docs).

**Gap to ideal:** add asymmetric workload identity or SPIFFE integration,
external secret management, TLS deployment guidance, rate limits, audit export,
policy bundles/versioning and end-to-end negative tests in a production
topology.

### 11. CI, release and supply-chain trust — 3/5

**Purpose:** make governance blocking and distribute a verifiable engine.

**Delivered now:** CI tests, installer E2E, docs build, composite action,
pre-commit hooks, native Git hooks, GoReleaser archives and SHA-256 checksums.

**Strength:** multiple adoption levels support gradual rollout.

**Gap to ideal:** replace `<owner>/<repo>` placeholders, publish the action,
sign binaries, publish SBOMs and SLSA provenance, add CodeQL/secret/dependency
scanning, pin actions by immutable digest where appropriate and run OpenSSF
Scorecard. [OpenSSF Scorecard](https://scorecard.dev/) explicitly evaluates
CI, review, token permissions, packaging and signed releases.

### 12. Import and adoption interoperability — 4/5

**Purpose:** let teams adopt POSE without discarding existing SDD work.

**Delivered now:** bounded, offline, dry-run imports for Spec Kit and OpenSpec;
symlink rejection; batch preflight; no overwrite; curation warnings.

**Strength:** POSE positions itself as a governance continuation rather than a
forced rewrite.

**Gap to ideal:** publish mapping fixtures, support custom source schemas through
plugins, preserve more provenance and consider a read-only diff/reconciliation
mode. Bidirectional sync should remain opt-in because two lifecycle authorities
create ambiguity.

### 13. Metrics and observability — 2/5

**Purpose:** show whether governance improves delivery outcomes and product
adoption.

**Delivered now:** pass/fail/partial history grouped by workflow, task or
context; privacy-preserving telemetry is opt-in and endpoint-less by default.

**Strength:** the data model is deterministic and privacy conservative.

**Gap to ideal:** correlate specs, commits, deployments and incidents; add the
five [DORA delivery metrics](https://dora.dev/guides/dora-metrics/) at the
application level; capture adoption, retention and task-success signals; emit
OpenTelemetry-compatible traces, metrics and logs for server operation. Avoid
using metrics as individual productivity targets.

### 14. Documentation, localization and diagnostics — 4/5

**Purpose:** shorten time to first governed delivery and make failures
actionable.

**Delivered now:** product README, MkDocs site, full CLI/manual references,
English and pt-BR scaffolds, JSON doctor output and explicit limitations.

**Strength:** operating docs are installed beside the code and remain editable.

**Gap to ideal:** publish stable URLs, add a complete download/install path,
remove release/action placeholders, add upgrade and troubleshooting guides,
test every documented command and add worked examples for brownfield and
monorepo adoption.

### 15. Extensibility and ecosystem — 2/5

**Purpose:** let teams add domains and integrations without maintaining a fork.

**Delivered now:** editable task maps, rules, workflows, skills, validation
checks and JSON projections.

**Strength:** many useful extensions require data changes, not Go code.

**Gap to ideal:** define versioned plugin manifests, compatibility constraints,
discovery, installation/update/removal, conflict handling, provenance and a
catalog. GitHub Spec Kit's current extensions, presets, workflows and bundles
provide a useful completeness benchmark for this layer.

### 16. Multi-repository and enterprise operation — 2/5

**Purpose:** preserve the same governance model across many teams and projects.

**Delivered now:** multi-root MCP selection, OPA hooks, execution identity and
Conductor reporter integration.

**Strength:** the open core already has clean composition boundaries.

**Gap to ideal:** centralized discovery, SSO/RBAC, tenant isolation, durable
orchestration, approvals, portfolio UX, policy distribution, retention and
support operations. These are the natural responsibilities of Crisol; keeping
them out of the local CLI preserves a credible freemium boundary.

## Priority improvement plan

### P0 — establish product trust and contract accuracy

1. Align binary, MCP and registry versions.
2. Resolve MCP catalog/schema drift and add exact conformance tests.
3. Replace public CI/action placeholders and document a real download path.
4. Dogfood POSE in this standalone repository with product-owned specs,
   reports and a roadmap instead of an empty instance.
5. Sign releases and publish SBOM plus SLSA provenance.
6. Add OpenSSF Scorecard, dependency, secret and static-analysis checks.

### P1 — deepen delivery value

1. Add requirement-to-check-to-commit traceability.
2. Export structured validation results as JSON, JUnit and SARIF where relevant.
3. Add changed-scope validation, per-check timeout and Harness isolation.
4. Add DORA-compatible delivery integrations and OpenTelemetry server signals.
5. Add follow-up ownership, SLA and recurrence-effectiveness views.
6. Expand the baseline stack catalog and publish monorepo recipes.

### P2 — build ecosystem and scale path

1. Publish a versioned extension/skill catalog with provenance.
2. Add optional semantic knowledge and follow-up adapters with human approval.
3. Add cross-repository references and portfolio projections.
4. Connect the open core to Crisol for durable orchestration, visual operation,
   centralized policy and team analytics.

## Reassessment protocol

Re-run this assessment at each minor release:

1. Pin the release commit and list evidence commands.
2. Score only behavior that is implemented and verified.
3. Link each score increase to a check, report or public artifact.
4. Record benchmark changes; external products evolve.
5. Keep historical assessments so product progress is visible.

Do not collapse the table into one percentage. The purpose is to choose the
next highest-value constraint, not to optimize a vanity score.
