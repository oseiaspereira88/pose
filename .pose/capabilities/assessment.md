---
schema_version: 1
assessed_at: 2026-07-22
baseline_commit: c9a08fa
method: local source inspection, delivered-spec Final Reports, pose doctor --json, pose assess, MCP tools/list golden fixture and repository checks
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
- evidence: spec:pose-package-manager-distribution, spec:pose-upgrade-compatibility-lab, spec:pose-doctor-guided-remediation, doc:docs-site/docs/capability-assessment.md
- gaps: winget-pkgs submission not yet made; Scoop and Nix channels uncovered

Two real package-manager channels; proven in-place upgrade against a populated instance.

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
- evidence: spec:pose-agent-skills-conformance
- gaps:

Agent Skills contract is a CI gate across all 9 skills, both locales.

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
- evidence: spec:pose-stack-catalog-expansion, spec:pose-structured-validation-results, spec:pose-changed-scope-validation, spec:pose-validation-runtime-guardrails, spec:pose-monorepo-validation-recipes
- gaps:

Python/.NET/monorepo stacks, JSON/JUnit/SARIF, timeouts and Harness isolation delivered.

## Mechanism: evidence-history-insights
- title: Evidence, history and insights
- score: 4
- target: 5
- evidence: spec:pose-requirement-evidence-traceability, spec:pose-release-signing, spec:pose-slsa-provenance, spec:pose-capability-mechanism
- gaps: provenance covers release artifacts, not per-spec closeout evidence; closeout actor identity is Git identity, not a captured structured field

Release artifacts are signed and capability evidence now has typed resolution
plus append-only snapshots; per-report actor attestation is still absent.

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
- evidence: spec:pose-mcp-catalog-conformance, spec:pose-mcp-project-scope-contract, spec:pose-mcp-protocol-completeness, spec:pose-capability-mechanism
- gaps:

Golden-fixture catalog conformance, uniform project scoping, pagination, 32 tools.

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
- evidence: spec:pose-release-signing, spec:pose-cyclonedx-sbom, spec:pose-slsa-provenance, spec:pose-ossf-security-baseline, spec:pose-reproducible-release-verification
- gaps: real N-minus-1 comparison in Verify release unexercised until a second real published release exists

Signed, SBOM'd, provenance-attested releases; security workflows green.

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
- evidence: spec:pose-localization-docs-contract, spec:pose-doctor-guided-remediation, spec:pose-capability-mechanism
- gaps:

Locale-parity bug fixed, self-inspecting docs tests, guided remediation and a
structured assessment as the source behind the narrative documentation.

## Mechanism: extensibility-ecosystem
- title: Extensibility and ecosystem
- score: 5
- target: 5
- evidence: spec:pose-extension-catalog-lifecycle
- gaps: community catalog still to populate (adoption gap, not mechanism gap)

Versioned manifest, install/list/remove/verify, provenance and revocation.

## Mechanism: multi-repo-enterprise
- title: Multi-repository and enterprise operation
- score: 4
- target: 5
- evidence: spec:pose-harne8-control-plane-integration, adr:2026-07-19-harne8-control-plane-composition-boundaries.md
- gaps: centralized discovery UX, SSO/RBAC, tenant isolation, durable orchestration and portfolio visualization are Harne8's responsibility by design

Harne8 boundary ratified and tested; the local engine deliberately stops here.
