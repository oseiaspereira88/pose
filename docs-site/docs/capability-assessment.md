# Capability assessment

**Doc type:** Explanation &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

**Assessment date:** 2026-07-19
**Evidence baseline:** commit `38a248d`, local source inspection, spec Final
Reports for all 35 delivered specs, `pose doctor --json`, MCP `tools/list`
golden fixture and repository checks
**Purpose:** measure fitness against POSE's own promises and relevant
best-of-breed practices, not manufacture a universal product ranking.

> **Structured source of truth:** the scores, evidence references and gaps
> in this document are maintained as a POSE-native artifact at
> `.pose/capabilities/assessment.md`, validated by `pose assess` (see the
> CLI reference). This prose remains the narrative; the artifact is the
> data.

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

Since the previous assessment (2026-07-18, commit `d9c0b98`), all seven
roadmaps in the product portfolio have shipped — 35 specs, each closed with
requirement trace, deterministic validation evidence and a Final Report.
Every item in that assessment's P0 and P1 improvement plan is delivered;
most of P2 is delivered as well, with one item (durable multi-team
orchestration, visual operation, tenant-scoped policy distribution) still
correctly out of scope for a local-first engine and living in Harne8 by
design.

POSE has moved from "credible open-source, single-repo governance engine
with a large distance to reference-grade" to a governance engine that is
reference-grade across most of its own claimed surface: distribution trust
(signed, SBOM'd, provenance-attested releases on two real package-manager
channels), delivery-outcome measurement (DORA + OpenTelemetry), MCP protocol
completeness, structured validation output, extension lifecycle and
cross-repository portfolio projection are no longer gaps. The remaining
distance is concentrated in a smaller set of things that were never fully
targeted by this portfolio: production-grade identity/secret management for
remote MCP deployments, a populated third-party extension ecosystem (the
lifecycle mechanism exists; the community catalog does not yet), and the
handful of specs whose own Final Reports flag a real follow-through step
still pending (WinGet's `winget-pkgs` submission, a first real
N-minus-1 `Verify release` run against a published tag).

| Mechanism | Current | Target | Summary |
|---|---:|---:|---|
| Install, upgrade and local-first runtime | 5 | 5 | Two real package-manager channels; proven in-place upgrade against a populated instance |
| Spec lifecycle and closeout | 5 | 5 | Requirement-to-check-to-commit trace and structured amendment history close the prior gap |
| Task routing, workflows, rules and skills | 5 | 5 | Agent Skills contract is now a CI gate across all 9 skills, both locales |
| Dependencies, readiness and roadmaps | 4 | 5 | Cross-repository portfolio projection with ownership/criticality; no capacity/time scheduling by design |
| Validation matrix and structural checks | 5 | 5 | Python/.NET/monorepo stacks, JSON/JUnit/SARIF, timeouts and Harness isolation all delivered |
| Evidence, history and insights | 4 | 5 | Requirement trace links checks to commits; release artifacts are signed, per-report evidence is not |
| Follow-ups and recurrence | 5 | 5 | Owner/SLA and measured intervention effectiveness close the prior gap |
| Knowledge governance | 4 | 5 | Usage traceability and explainable semantic-advisory retrieval delivered; RBAC mapping still open |
| MCP and agent interoperability | 5 | 5 | Golden-fixture catalog conformance, uniform project scoping, pagination, 30 tools |
| Policy, identity and audit | 4 | 5 | Identity-gated validation orchestration and bounded audit fields; SPIFFE/secret-mgmt/TLS still external |
| CI, release and supply-chain trust | 5 | 5 | Signed, SBOM'd, provenance-attested releases; CodeQL/govulncheck/gitleaks/Scorecard all green |
| Import and adoption interoperability | 4 | 5 | Three executable, end-to-end-tested brownfield kits; no plugin-based custom source schemas yet |
| Metrics and observability | 5 | 5 | All five DORA metrics and OTel traces/metrics for server operation; log export awaits OTel Logs SDK stability |
| Documentation, localization and diagnostics | 5 | 5 | Locale-parity bug fixed, self-inspecting docs tests, guided remediation, docs-as-tests monorepo recipes |
| Extensibility and ecosystem | 5 | 5 | Versioned manifest, install/list/remove/verify, provenance and revocation; community catalog still to populate |
| Multi-repository and enterprise operation | 4 | 5 | Harne8 boundary ratified and tested; durable orchestration/tenancy remain Harne8's job by design |

## Detailed findings

### 1. Install, upgrade and local-first runtime — 5/5

**Purpose:** make governance easy to adopt without creating a hosted-service
dependency.

**Delivered now:** a deterministic Homebrew formula and WinGet manifest
generator (`pose release-package-manifests`) driven by the release's own
checksums, wired strictly after every release verification step; a clean-host
CI matrix installing/doctoring/uninstalling through both channels on macOS and
Windows; `pose doctor --fix` (preview) and `--fix --yes` covering three
confined, idempotent repairs (pre-commit hook, `.mcp.json`, Claude skill
symlinks) reusing already-tested code paths; full unit coverage of
`cmdUpgrade` against a populated instance (real spec, real knowledge note, a
hand-edited managed file) proving dry-run non-mutation, schema-only apply and
idempotent reapply; a symlink-escape gap in managed directories closed via
`ensureManagedDirSafe`.

**Strength:** installation and upgrade are no longer just tested in the
abstract — they are proven against instances that already have real content
in them, which is the scenario that actually breaks upgrade paths in
practice.

**Gap to ideal:** the WinGet manifest generator is real and CI-tested, but
publication into `winget-pkgs` itself is a maintainer-reviewed submission
that has not yet been made; Scoop and Nix channels remain uncovered. Neither
is a mechanism gap — both are real follow-through steps documented as open
in `pose-package-manager-distribution`'s own Final Report.

### 2. Spec lifecycle and closeout — 5/5

**Purpose:** prevent ambiguous work from starting and incomplete work from
being declared done.

**Delivered now:** an in-spec requirement trace contract (grammar, parser,
lint gates, metrics) with a bidirectional MCP projection
(`pose_requirement_trace`) linking stable requirement IDs to the checks,
results and commits that satisfied them; an append-only amendment contract
(`amendments.jsonl`) with a `pose amend` command distinguishing material
changes from editorial acknowledgment, a deterministic hash-based closeout
gate, and an MCP projection (`pose_spec_amendments`).

**Strength:** POSE now closes the two gaps this document previously named
explicitly — requirement-to-test/result traceability and structured change
history for spec amendments — without weakening the human-review requirement
for intent changes.

**Gap to ideal:** approval records remain a human act (a reviewer approving a
PR), not a machine-readable signed attestation of who approved what spec
change. This is a deliberate boundary, not an oversight — POSE's own model
treats human judgment on intent as non-negotiable.

**Benchmark:** [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)
offers a rich SDD lifecycle, extensions, presets and workflows;
[OpenSpec](https://github.com/Fission-AI/OpenSpec) is strong at lightweight
brownfield change deltas and archive-to-source flow; [Kiro](https://aws.amazon.com/documentation-overview/kiro/)
integrates specs, steering and hooks into an agentic development service.

### 3. Task routing, workflows, rules and skills — 5/5

**Purpose:** translate task intent into the correct procedure, constraints and
checks across agents.

**Delivered now:** compatibility frontmatter (`pose_schema_range`, `clients`,
`capabilities`) on all 9 shipped skills in both locales; `pose skills-check`
as a CI gate; `pose_skills_check` as an MCP tool; broken-link detection caught
one real defect during delivery, proving the gate exercises real content, not
a placeholder check.

**Strength:** the full [Agent Skills](https://agentskills.io/specification)
contract is now continuously validated rather than assumed, and routing
remains data — rules can evolve without copying every workflow.

**Gap to ideal:** a signed, versioned catalog with update/conflict handling
for skills is now the generic extension-lifecycle mechanism described in
finding 15, not a gap specific to this mechanism.

### 4. Dependencies, readiness and roadmaps — 4/5

**Purpose:** start only eligible work and expose planning order.

**Delivered now:** `pose portfolio-projection` reconciles dependency,
readiness, ownership and criticality across repositories, reusing the MCP
server's own project-authorization boundary; an additive `xref:` reference
grammar; explicit `blocked`/`stale`/`unauthorized`/`unknown` resolution
states so a reader always knows when a projection is out of date, persisted
as a revisioned, tombstoned report.

**Strength:** readiness and cross-repository planning now derive from
versioned contracts and an explicit staleness signal, not board status or a
manually maintained spreadsheet.

**Gap to ideal:** there is still no graphical impact/critical-path
visualization or capacity/time-allocation view — deliberately: POSE stays a
projection source, not a transactional scheduler. That visualization layer
is Harne8's Portal, not the local engine's job.

### 5. Validation and structural integrity — 5/5

**Purpose:** replace subjective "looks done" decisions with repeatable checks.

**Delivered now:** a maintained stack catalog now covering Node.js, Go, Rust,
Java, Python (five package managers) and .NET, plus a read-only `pose stacks`
detection command; a canonical, versioned validation result model with
additive `--json`/`--junit`/`--sarif` emission; deterministic changed-scope
selection (`--changed-from/--changed-to`) over declared dependency edges with
a safe-execution fallback and an `--explain` trace; per-check timeout and
output-ceiling guardrails with process-group cancellation, plus an
`isolation: "required"` classification for checks that must run in the
Harness rather than locally; three docs-as-tests monorepo recipes (JS
workspace, declared dependency graph, mixed-language with a shared
high-criticality module) executed by a real test, not just documented.

**Strength:** every gap this document previously named by name — additional
stacks, standard result formats, timeouts/isolation, changed-scope selection,
monorepo recipes — is closed with executable evidence, not aspiration.

**Gap to ideal:** none identified against this mechanism's original scope.
Future stack additions (e.g. Ruby, PHP) remain a routine catalog extension,
not a structural gap.

### 6. Evidence, history and insights — 4/5

**Purpose:** preserve why a delivery was accepted and feed future decisions.

**Delivered now:** requirement trace (finding 2) links requirement IDs to the
checks and commits that satisfied them; release artifacts carry SLSA
provenance, CycloneDX SBOMs and Sigstore signatures (finding 11); Markdown
reports, append-only JSONL history and MCP insights remain inspectable with
ordinary Git.

**Strength:** the traceability gap this document previously called out by
name — "link requirement IDs to checks and commits" — is closed, and release
evidence now carries real cryptographic provenance, not just a stable-field
hash.

**Gap to ideal:** that provenance chain covers release artifacts, not
individual spec closeout evidence — a `spec.md`'s Final Report and its
validation history are still integrity-protected by Git history alone, not
independently signed or attested per closeout. Actor/approval identity for a
spec closeout is still whoever's Git identity signed the merge commit, not a
captured, structured field.

**Benchmark:** [SLSA](https://slsa.dev/spec/v1.0/levels) defines progressive
build provenance guarantees, now applied to every POSE release artifact.
CycloneDX defines interoperable SBOM media types and predicates, also now
applied. Per-spec evidence signing would be a POSE-specific extension of the
same idea, not something either standard covers directly.

### 7. Follow-ups and recurrence — 5/5

**Purpose:** stop residual work and repeated failures from disappearing into
free text.

**Delivered now:** an inline ownership/SLA contract for follow-ups (syntax,
parser, closeout gate, projections, opt-in blocking) with the live backlog
migrated to owned entries; an intervention registry
(`interventions.jsonl`) feeding a deterministic before/after
effectiveness projection with verdicts and explicit data-quality warnings,
plus opt-in duration/cost telemetry.

**Strength:** POSE can now answer not just "what follow-ups are open" but
"who owns them, by when" and "did the systemic fix we shipped actually
reduce recurrence" — closing this document's two previously named gaps in
full.

**Gap to ideal:** none identified against this mechanism's original scope.

### 8. Operational knowledge — 4/5

**Purpose:** preserve reusable context without creating permanent, ownerless
documentation.

**Delivered now:** a stable `knowledge:<slug>` citation contract with a
dangling-reference gate; a derived usage projection
(`knowledge-usage`) honoring TTL immutability, so a reader can see when
governed knowledge actually influenced work rather than trusting that it was
read; deterministic, explainable, sensitivity-filtered advisory retrieval
(`knowledge-suggest`) with pre-retrieval scoping.

**Strength:** knowledge now has both decay/accountability by design (prior
assessment) and consumption traceability (this delivery) — the two halves of
"does this knowledge base actually get used" are both answered.

**Gap to ideal:** retrieval is lexical similarity over already-tested
primitives, not embedding/LLM-based semantic search, and there is still no
external identity/RBAC mapping for who is allowed to see sensitive knowledge
beyond POSE's own sensitivity field.

### 9. MCP and agent interoperability — 5/5

**Purpose:** expose governance consistently to any MCP-capable agent.

**Delivered now:** a versioned golden catalog fixture
(`testdata/tool-catalog.golden.json`) freezing the exact `tools/list` payload
— 30 tools — with bijection and negative-path tests against source registry,
runtime and documentation; a uniform `project_id` schema across every
multi-root tool with typed `ProjectUnknownError`/`ProjectAmbiguousError` and
structured `error_code` fields; opaque cursor pagination
(`cursor`/`limit`/`next_cursor`) on all four `pose_list_*` tools, fully
additive; a deterministic tie-order fix in `ListReports`.

**Strength:** the catalog drift this document previously flagged by name
(`pose_validate` in the ADR vs. no matching tool) is structurally impossible
to reintroduce now — the golden fixture and bijection tests fail CI the
moment source, runtime and docs disagree.

**Gap to ideal:** none identified against this mechanism's original scope.
Resources/prompts remain unused because no current governance use case needs
them, consistent with the prior assessment's own caution against
overbuilding.

**Benchmark:** the [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
defines schema-based discovery, pagination and invocation and recommends
human control over tool use — POSE's catalog now demonstrably conforms.

### 10. Policy, identity and audit — 4/5

**Purpose:** authorize every remote governance call within project and run
scope.

**Delivered now:** a digest-pinned, identity-gated, idempotent
validation-orchestration state machine
(`pose_validate_request/approve/submit/status/cancel`) requiring explicit
approval before untrusted execution; audit log fields bounded against
oversized caller-supplied identity values (`truncateForAudit`), closing a
log-volume-abuse surface without touching the auditor's documented purpose
of recording who invoked what.

**Strength:** the request/approve/submit split gives policy a real
intervention point before execution, not just an allow/deny log line after
the fact.

**Gap to ideal:** asymmetric workload identity or SPIFFE integration,
external secret management, TLS deployment guidance, rate limits, audit
export and policy bundle/versioning remain entirely unaddressed — no roadmap
in this portfolio targeted them. This is the mechanism with the largest gap
remaining relative to its own prior assessment.

### 11. CI, release and supply-chain trust — 5/5

**Purpose:** make governance blocking and distribute a verifiable engine.

**Delivered now:** keyless Sigstore signing of every release artifact with
offline-verifiable bundles and a pinned issuer/identity policy; per-archive
CycloneDX SBOMs with schema and direct-dependency validation gates; SLSA v1
provenance attestation with a stated Build L2 claim and documented L3
limitation; an independent `Verify release` workflow running in a clean
environment with layered authentication (checksum, signature, SBOM,
provenance) before any execution; a versioned `compatibility.json`
(engine/schema/upgrade pairs, support policy) generating a
`compatibility-report.md` gate in CI; a placeholder-free public install
contract for Linux/macOS/Windows; the security workflow (CodeQL, govulncheck,
gitleaks, dependency review) and an OpenSSF Scorecard workflow, both
confirmed green this cycle after fixing a stale Go toolchain (govulncheck),
a test-fixture false positive (gitleaks) and a CodeQL build-mode
misconfiguration (switched to `autobuild`).

**Strength:** every gap this document previously named by name — signatures,
SBOM, provenance, Scorecard, dependency/secret/static-analysis scanning,
public placeholders — is closed with running, green CI evidence, not a
documented intention.

**Gap to ideal:** the reproducible-release-verification spec's own Final
Report notes its real N-minus-1 comparison path remains unexercised until a
real prior published release exists to compare against — the mechanism is
built and tested against synthetic fixtures, but its first real production
run is still pending. [OpenSSF Scorecard](https://scorecard.dev/) results are
now published and visible; further score improvement is routine maintenance,
not a structural gap.

### 12. Import and adoption interoperability — 4/5

**Purpose:** let teams adopt POSE without discarding existing SDD work.

**Delivered now:** three checked-in, executable adoption kits
(`examples/brownfield-kits/{direct-adoption,spec-kit-import,openspec-import}/`)
with staged visibility→adoption→blocking-gate READMEs, each verified
end-to-end by a dedicated test against a real, intentionally-imperfect
fixture (not a pristine synthetic one).

**Strength:** the prior gap — "publish mapping fixtures" — is closed with
real, tested, checked-in examples rather than documentation prose, and the
kits deliberately preserve real migration friction instead of hiding it.

**Gap to ideal:** custom source schemas through a plugin mechanism and a
read-only diff/reconciliation mode remain open. Bidirectional sync
deliberately remains out of scope — two lifecycle authorities create
ambiguity POSE should not resolve implicitly.

### 13. Metrics and observability — 5/5

**Purpose:** show whether governance improves delivery outcomes and product
adoption.

**Delivered now:** `pose record-deployment`/`record-incident` for
quality-gated event ingestion; `pose dora-metrics` computing all five current
DORA metrics with a three-state result and a documented Reliability proxy;
`pose adoption-metrics` deriving activation, time-to-first-gate, retention
and task-success from existing spec/history data; `pose events-housekeeping`
for retention; opt-in OpenTelemetry traces and metrics (stable SDK,
OTLP/HTTP) plus a trace-correlated, redacted structured logger wired through
every MCP `tools/call`, with graceful startup/shutdown including
SIGINT/SIGTERM handling.

**Strength:** both gaps this document previously named explicitly by
standard name — "the five DORA delivery metrics" and "OpenTelemetry-compatible
traces, metrics and logs" — are delivered, and the two metric families are
kept in genuinely separate reports so adoption data is never blended with
DORA numbers into a false causal claim.

**Gap to ideal:** logs are a local structured writer, not yet OTLP-exported —
deferred deliberately until the OTel Logs SDK reaches a stable release, per
the delivering spec's own documented risk. DORA event ingestion is manual or
CI-driven with no automatic collector, a stated scope boundary rather than an
oversight.

### 14. Documentation, localization and diagnostics — 5/5

**Purpose:** shorten time to first governed delivery and make failures
actionable.

**Delivered now:** a real, previously undetected locale-parity bug fixed
(English default templates that were actually Portuguese, with no `pt-BR`
translation on file); a self-inspecting documented-commands contract test
that reads the CLI's own dispatch table instead of duplicating it; all 12
docs-site pages classified by Diátaxis type with a visible
version-applicability line; a docs security scan reusing the skills
conformance patterns; `pose doctor --fix` turning diagnosable failures into
guided, confined, idempotent remediation instead of prose instructions.

**Strength:** documentation now has the same kind of executable guarantee
POSE demands of everything else — a doc claiming a command exists is
verified against the CLI's real dispatch table, not maintained by hand.

**Gap to ideal:** none identified against this mechanism's original scope.

### 15. Extensibility and ecosystem — 5/5

**Purpose:** let teams add domains and integrations without maintaining a
fork.

**Delivered now:** an `extension.json` manifest contract with path
confinement, permission whitelisting, revocation and provenance; `pose
extension install/list/remove/verify` — dry-runnable, consent-gated,
transactional.

**Strength:** every element this document previously named as the completion
benchmark — versioned manifests, compatibility constraints, discovery,
install/update/removal, conflict handling, provenance, a catalog mechanism —
is delivered.

**Gap to ideal:** the lifecycle mechanism is complete; a populated
third-party or community extension catalog is not — this is an ecosystem
adoption gap (will grow with usage), not a missing mechanism. GitHub Spec
Kit's populated extensions/presets/bundles remain the useful comparison for
that adoption curve, not for the mechanism itself.

### 16. Multi-repository and enterprise operation — 4/5

**Purpose:** preserve the same governance model across many teams and
projects.

**Delivered now:** a ratified five-component responsibility table mapping
POSE/Conductor/Harness/GraphForge/Portal to already-tested surface;
`pose reconcile-evidence`, an identity-bound, append-only Harness-result
reconciliation contract rejecting silent mutation with tenant-scoped
retention; an executable end-to-end test proving offline degradation (not
just a documentation claim) when Harne8 is absent; cross-repository
portfolio projection (finding 4); advisory semantic suggestions across
knowledge, follow-ups and recurrence patterns with rationale and sensitivity
filtering (`pose semantic-suggest`).

**Strength:** the open-core/Harne8 boundary is no longer just an
architectural intention — it is a ratified, citable table plus a tested
contract proving the core keeps working when Harne8 is not present.

**Gap to ideal:** centralized discovery UX, SSO/RBAC, tenant isolation,
durable orchestration, approvals, portfolio visualization, policy
distribution and support operations remain Harne8's responsibility by
design, not the local engine's. Keeping them out of the CLI is the point,
not a shortfall — but it is also why this mechanism cannot honestly score
higher from pose-dist's side alone.

## Priority improvement plan

Every P0 and P1 item from the 2026-07-18 assessment is delivered; the P2 item
this document previously left explicitly to Harne8 (durable orchestration,
visual operation) is unchanged as an intentional boundary. The plan below
covers what is genuinely still open, drawn from the residual risks and
follow-ups each delivering spec recorded in its own Final Report.

### P0 — close real, spec-identified follow-through gaps

1. Submit the generated WinGet manifest to `winget-pkgs` (the generator and
   CI matrix are done; the maintainer-reviewed submission is not).
2. Exercise a real N-minus-1 comparison in `Verify release` / `compat.sh`
   once a second real published release exists — the synthetic-fixture path
   is fully tested, the real-history path is not yet.

### P1 — harden what no roadmap in this portfolio targeted

1. Add workload identity (SPIFFE or equivalent), external secret management
   and TLS deployment guidance for remote MCP deployments.
2. Add rate limiting, audit export and policy bundle/versioning to the
   OPA-based policy layer.
3. Add a plugin mechanism for custom import source schemas and a read-only
   diff/reconciliation mode for already-adopted repositories.
4. Add OTLP log export once the OpenTelemetry Logs SDK reaches a stable
   release (currently a documented, deliberate wait, not an open task).

### P2 — grow ecosystem adoption on top of delivered mechanisms

1. Seed the extension catalog with real third-party or community entries —
   the install/update/removal/provenance mechanism is done; population is
   not.
2. Evaluate a real embedding/LLM-backed provider for
   `pose semantic-suggest` where one can be safely configured and tested,
   reducing (not eliminating) the residual risk its own spec names for
   lexical similarity.
3. Add Scoop and Nix package-manager channels alongside the delivered
   Homebrew/WinGet pair.
4. Continue building Harne8's durable orchestration, visual portfolio
   operation, tenant-scoped policy distribution and support tooling on top
   of the now-ratified open-core boundary and `pose reconcile-evidence`
   contract — this remains Harne8's roadmap, not pose-dist's.

## Reassessment protocol

Re-run this assessment at each minor release:

1. Pin the release commit and list evidence commands.
2. Score only behavior that is implemented and verified.
3. Link each score increase to a check, report or public artifact.
4. Record benchmark changes; external products evolve.
5. Keep historical assessments so product progress is visible.

Do not collapse the table into one percentage. The purpose is to choose the
next highest-value constraint, not to optimize a vanity score.
