---
schema_version: 1
assessed_at: 2026-08-17
baseline_commit: dbf5b77
method: v1.4.3 published-release source inspection, delivered-spec Final Reports, four real cut/published/independently-verified releases (v1.4.0-v1.4.3), pose doctor --json, pose assess, MCP tools/list golden fixture and repository checks
---

# Capability assessment

Structured migration of `docs-site/docs/capability-assessment.md`
(2026-07-19). Scores are human judgment on a 0-5 scale; the target is not
always 5. The prose document remains the narrative; this artifact is the
structured source of truth for scores, evidence and gaps.

## Mechanism: install-upgrade-runtime
- title: Install, upgrade and local-first runtime
- score: 5
- target: 5
- evidence: spec:pose-package-manager-distribution, spec:pose-upgrade-compatibility-lab, spec:pose-doctor-guided-remediation, spec:pose-upgrade-path-audit-fixes, spec:pose-update-instance-directory-completeness, spec:pose-derived-index-self-referential-leak, doc:docs-site/docs/capability-assessment.md
- gaps: winget-pkgs submission not yet made; Scoop and Nix channels uncovered

Two real package-manager channels; proven in-place upgrade against a populated
instance. A field audit of `pose update`/`pose install` against seven real,
independently-owned repositories (not synthetic fixtures) surfaced and closed
11 correctness/robustness defects: locale handling was unreliable without
`--force`, a plain update could leave an old instance with broken references
undetected by `pose doctor` (now two new doctor checks), a hand-edit outside
an instance-owned AGENTS.md/POSE.md section could be silently dropped (now
warned and consistently backed up), and the embedded scaffold leaked this
product's own computed graph/module data into fresh instances (closed the
same self-referential-leak class already fixed for policy files under issue
#17). Verified against real upgrades from four prior published releases
(1.1.0, 1.0.0, 0.19.0, 0.18.2) on every one of four consecutive real
releases cut this cycle (v1.4.0-v1.4.3).

## Mechanism: spec-lifecycle-closeout
- title: Spec lifecycle and closeout
- score: 5
- target: 5
- evidence: spec:pose-requirement-evidence-traceability, spec:pose-spec-amendment-history
- gaps: approval records are human acts, not machine-readable signed attestations (deliberate boundary)

Requirement-to-check-to-commit trace and structured amendment history.

## Mechanism: task-routing-workflows-skills
- title: Task routing, workflows, rules and skills
- score: 5
- target: 5
- evidence: spec:pose-agent-skills-conformance, spec:pose-component-aware-review-plans
- gaps:

Agent Skills contract is a CI gate across all 11 skills, both locales; review
plans now compose component-specific policy and safe tool guidance.

## Mechanism: dependencies-readiness-roadmaps
- title: Dependencies, readiness and roadmaps
- score: 4
- target: 5
- evidence: spec:pose-cross-repo-portfolio, adr:2026-07-19-cross-repo-portfolio-reuses-mcp-project-authorization.md
- gaps: no graphical impact/critical-path visualization (Harne8 Portal's job by design); no capacity/time scheduling by design

Cross-repository portfolio projection with ownership/criticality.

## Mechanism: validation-structural-integrity
- title: Validation matrix and structural checks
- score: 5
- target: 5
- evidence: spec:pose-stack-catalog-expansion, spec:pose-structured-validation-results, spec:pose-changed-scope-validation, spec:pose-validation-runtime-guardrails, spec:pose-monorepo-validation-recipes, spec:pose-validation-scanner-consolidation, spec:pose-upgrade-path-audit-fixes, spec:pose-discovery-gitignore-and-root-alias-fix
- gaps:

Python/.NET/monorepo stacks, JSON/JUnit/SARIF, timeouts and Harness isolation
delivered. Stack discovery is now a single shared detector across `pose
index`/`validate`/`install`/`init` (previously two independently-drifting
scanners), classifies Android/Kotlin Gradle modules distinctly from generic
JVM ones, excludes synthetic testdata/fixture content, and — found live on a
real repository whose vendored subtree was entirely gitignored — now respects
`.gitignore` instead of walking excluded trees as if they were the project's
own deliverable modules.

## Mechanism: evidence-history-insights
- title: Evidence, history and insights
- score: 4
- target: 5
- evidence: spec:pose-requirement-evidence-traceability, spec:pose-release-signing, spec:pose-slsa-provenance, spec:pose-capability-mechanism, spec:pose-review-bundle-convergence
- gaps: cryptographic provenance covers release artifacts and optional external review envelopes, not every local report or closeout event

Release artifacts are signed, capability evidence has typed resolution and
append-only snapshots, and review approvals bind an immutable semantic bundle;
ordinary local reports remain unsigned.

## Mechanism: followups-recurrence
- title: Follow-ups and recurrence
- score: 5
- target: 5
- evidence: spec:pose-followup-ownership-sla, spec:pose-recurrence-effectiveness
- gaps:

Owner/SLA and measured intervention effectiveness.

## Mechanism: operational-knowledge
- title: Knowledge governance
- score: 4
- target: 5
- evidence: spec:pose-knowledge-consumption-traceability, spec:pose-semantic-governance-assist
- gaps: retrieval is lexical, not embedding/LLM-based; no external identity/RBAC mapping for sensitive knowledge

Usage traceability and explainable semantic-advisory retrieval delivered.

## Mechanism: mcp-agent-interop
- title: MCP and agent interoperability
- score: 5
- target: 5
- evidence: spec:pose-mcp-catalog-conformance, spec:pose-mcp-project-scope-contract, spec:pose-mcp-protocol-completeness, spec:pose-capability-mechanism, spec:pose-component-aware-review-plans, spec:pose-review-bundle-convergence
- gaps:
- paths: pose-mcp/internal/mcpserver/*.go

Golden-fixture catalog conformance, uniform project scoping, pagination and 50 tools.

## Mechanism: policy-identity-audit
- title: Policy, identity and audit
- score: 4
- target: 5
- evidence: spec:pose-safe-validate-orchestration
- gaps: SPIFFE/workload identity, external secret management, TLS deployment guidance, rate limits, audit export and policy bundle/versioning unaddressed

Identity-gated validation orchestration and bounded audit fields.

## Mechanism: ci-release-supply-chain
- title: CI, release and supply-chain trust
- score: 5
- target: 5
- evidence: spec:pose-release-signing, spec:pose-cyclonedx-sbom, spec:pose-slsa-provenance, spec:pose-ossf-security-baseline, spec:pose-reproducible-release-verification, spec:pose-upgrade-path-audit-fixes, spec:pose-discovery-gitignore-and-root-alias-fix
- gaps:

Signed, SBOM'd, provenance-attested releases; security workflows green. The
prior gap ("real N-minus-1 comparison unexercised until a second real
published release exists") is closed: four consecutive real releases were
cut, published and independently verified this cycle (v1.4.0 through v1.4.3),
each with a bit-identical independent rebuild and each exercising real
upgrade paths from four previously-published versions (1.1.0, 1.0.0, 0.19.0,
0.18.2) against pinned, checksum-authenticated prior artifacts — not
simulated. Three of the four releases (v1.4.1-v1.4.3) shipped same-day
follow-up fixes for defects found auditing the immediately preceding release
against real, independently-owned external repositories, which is itself
evidence the release-to-fix-to-release loop works end to end under real
conditions, not only in isolation.

## Mechanism: import-adoption-interop
- title: Import and adoption interoperability
- score: 4
- target: 5
- evidence: spec:pose-brownfield-reference-kits
- gaps: no plugin mechanism for custom source schemas; no read-only diff/reconciliation mode

Three executable, end-to-end-tested brownfield kits.

## Mechanism: metrics-observability
- title: Metrics and observability
- score: 5
- target: 5
- evidence: spec:pose-dora-adoption-metrics, spec:pose-otel-observability
- gaps: log export awaits OTel Logs SDK stability (deliberate wait); DORA ingestion is manual/CI by scope

All five DORA metrics and OTel traces/metrics for server operation.

## Mechanism: docs-localization-diagnostics
- title: Documentation, localization and diagnostics
- score: 5
- target: 5
- evidence: spec:pose-localization-docs-contract, spec:pose-doctor-guided-remediation, spec:pose-capability-mechanism, spec:pose-onboarding-context-extraction, spec:pose-upgrade-path-audit-fixes
- gaps:

Locale-parity bug fixed, self-inspecting docs tests, guided remediation and a
structured assessment as the source behind the narrative documentation.
`pose init`/`pose install` now excerpts a brownfield target's own
`README.md`/`CLAUDE.md` into AGENTS.md's "Project context" on first install
instead of a generic placeholder. A real audit found and fixed a second class
of locale defect beyond the original parity bug: an explicit `--locale`
request could be silently ignored or produce a duplicated (not switched)
manual on an already-installed instance; both are now fixed. The same audit
also found `pose check`'s own summary line was unconditionally Portuguese
regardless of locale for every direct invocation, not only the one internal
call the fix originally targeted — now corrected.

## Mechanism: extensibility-ecosystem
- title: Extensibility and ecosystem
- score: 5
- target: 5
- evidence: spec:pose-extension-catalog-lifecycle, spec:pose-extension-catalog-resolution, spec:pose-rule-extension-locale-parity
- gaps: community catalog still to populate (adoption gap, not mechanism gap); only `pose-rule-kubernetes` is actually signed/published this way today, `pose-rule-backend-go`/`pose-rule-frontend-react` remain local-directory-installable only

Versioned manifest, install/list/remove/verify, provenance and revocation.
`pose extension install <extension-id>` now also resolves the ID against the
latest published GitHub release's signed assets directly — no local
directory required — and accepts a `--locale` flag, with pt-BR variants now
shipped for `pose-rule-backend-go`/`pose-rule-frontend-react`.

## Mechanism: multi-repo-enterprise
- title: Multi-repository and enterprise operation
- score: 4
- target: 5
- evidence: spec:pose-harne8-control-plane-integration, adr:2026-07-19-harne8-control-plane-composition-boundaries.md
- gaps: centralized discovery UX, SSO/RBAC, tenant isolation, durable orchestration and portfolio visualization are Harne8's responsibility by design

Harne8 boundary ratified and tested; the local engine deliberately stops here.
