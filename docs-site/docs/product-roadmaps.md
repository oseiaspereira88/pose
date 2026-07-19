# Product roadmaps

**Doc type:** Explanation &nbsp;·&nbsp; **Applies to:** POSE ≥ 0.9.0

**Planning baseline:** 2026-07-18  
**Canonical execution artifacts:** `.pose/roadmaps/*.md` and `.pose/specs/*/spec.md`

This portfolio converts the [capability assessment](capability-assessment.md)
into 7 governed roadmaps and 35 implementation specs. Dates are planning
targets, not delivery claims. A capability advances only when its spec crosses
the documented validation and closeout gates.

## Prioritization model

Order work using four filters, in this order:

1. **Critical absence:** address security, incorrect public contracts and
   release trust before convenience features.
2. **Dependency leverage:** prefer work that unlocks multiple later specs.
3. **User value:** reduce adoption friction and strengthen POSE's closed-loop
   governance differentiators.
4. **Execution risk:** prove contracts and fixtures before optimizing or scaling.

The integer `priority` in each spec is a global preference: lower means earlier.
`depends_on` remains the hard eligibility rule. Teams may parallelize eligible
specs but must not bypass dependencies to satisfy a target date.

## Portfolio sequence

| Order | Roadmap | Primary gap | Target window | Release-level outcome |
|---:|---|---|---|---|
| 1 | Product integrity | Contract drift and empty dogfood evidence | 2026-07-20 → 2026-08-28 | Public CLI, MCP, install and version claims agree |
| 2 | Supply-chain trust | Unsigned, unattested releases | 2026-08-03 → 2026-09-18 | Verifiable identity, SBOM, provenance and hardened CI |
| 3 | Governance traceability | Weak requirement-to-evidence links | 2026-08-24 → 2026-11-06 | Auditable intent-to-closure chain |
| 4 | Validation platform | Narrow output and runtime controls | 2026-08-24 → 2026-11-20 | Structured, bounded, polyglot validation |
| 5 | Agent interoperability | MCP drift and manual extension lifecycle | 2026-09-21 → 2026-12-18 | Conformant project-safe protocol and ecosystem |
| 6 | Adoption and DX | Distribution and onboarding friction | 2026-09-21 → 2027-01-29 | Trusted install, guided remediation and adoption kits |
| 7 | Insights and scale | No outcome integrations or portfolio layer | 2026-11-02 → 2027-03-31 | OTel/DORA signals and Harne8 composition |

## Coverage of the assessed mechanisms

| Assessed mechanism | Owning roadmap | Principal specs |
|---|---|---|
| Install, upgrade and local runtime | Adoption and DX | `pose-public-install-contract`, `pose-package-manager-distribution`, `pose-upgrade-compatibility-lab` |
| Spec lifecycle and closeout | Governance traceability | `pose-requirement-evidence-traceability`, `pose-spec-amendment-history` |
| Task routing, workflows, rules and skills | Agent interoperability | `pose-agent-skills-conformance`, `pose-extension-catalog-lifecycle` |
| Dependencies, readiness and roadmaps | Insights and scale | `pose-cross-repo-portfolio` |
| Validation and structural integrity | Validation platform | all five validation-platform specs |
| Evidence, history and insights | Governance traceability | `pose-requirement-evidence-traceability`, `pose-recurrence-effectiveness` |
| Follow-ups and recurrence | Governance traceability | `pose-followup-ownership-sla`, `pose-recurrence-effectiveness` |
| Operational knowledge | Governance traceability | `pose-knowledge-consumption-traceability` |
| MCP and agent interoperability | Product integrity / Agent interoperability | `pose-mcp-catalog-conformance`, `pose-mcp-project-scope-contract`, `pose-mcp-protocol-completeness` |
| Policy, identity and audit | Agent interoperability / Insights and scale | `pose-safe-validate-orchestration`, `pose-harne8-control-plane-integration` |
| CI, release and supply-chain trust | Supply-chain trust | all five supply-chain specs |
| Import and adoption interoperability | Adoption and DX | `pose-brownfield-reference-kits`, `pose-extension-catalog-lifecycle` |
| Metrics and observability | Insights and scale | `pose-otel-observability`, `pose-dora-adoption-metrics` |
| Documentation, localization and diagnostics | Adoption and DX | `pose-doctor-guided-remediation`, `pose-localization-docs-contract` |
| Extensibility and ecosystem | Agent interoperability | `pose-agent-skills-conformance`, `pose-extension-catalog-lifecycle` |
| Multi-repository and enterprise operation | Insights and scale | `pose-cross-repo-portfolio`, `pose-harne8-control-plane-integration` |

## Wave 0 — restore product truth

Ship product-integrity work before announcing broader release or ecosystem
maturity. The highest-risk gaps are inconsistent version sources, MCP catalog
drift, placeholder install paths and lack of standalone dogfooding.

**Promotion gate:** `pose check --strict`, MCP golden catalog tests, clean-host
install verification and a generated compatibility report pass for the same
release candidate.

## Wave 1 — establish trust and hard evidence

Run supply-chain work alongside the foundations of traceability and structured
validation. This makes the binary verifiable and turns results into reusable
contracts for CI, MCP and later analytics.

**Promotion gate:** consumers verify signatures and provenance; every result
has a stable schema; security workflows run with minimum permissions.

## Wave 2 — multiply governance value

Add requirement evidence, amendment history, owned follow-ups, recurrence
effectiveness, changed-scope validation and bounded execution. These features
reinforce POSE's core promise rather than merely matching an SDD authoring tool.

**Promotion gate:** at least two reference repositories demonstrate complete
intent → validation → evidence → follow-up → recurrence/knowledge traversal.

## Wave 3 — expand interoperability and adoption

Complete project-scoped MCP behavior, Agent Skills conformance, signed
extensions, package channels and brownfield kits.

**Promotion gate:** independent clients and clean environments pass published
compatibility suites without privileged manual setup.

## Wave 4 — measure outcomes and scale through Harne8

Add OpenTelemetry signals, DORA-compatible integrations, semantic assist with
human confirmation, cross-repository projections and control-plane composition.

**Promotion gate:** multi-repository pilots prove tenant isolation, policy,
retention and offline degradation while showing team-level delivery outcomes.

## Benchmark references

- Use the [MCP 2025-06-18 specification](https://modelcontextprotocol.io/specification/2025-06-18/)
  for protocol contracts.
- Use the [Agent Skills specification](https://agentskills.io/specification) for portable skills.
- Use [SLSA 1.2](https://slsa.dev/spec/v1.2/),
  [CycloneDX](https://cyclonedx.org/specification/overview/),
  [Sigstore](https://docs.sigstore.dev/) and
  [OpenSSF Scorecard](https://scorecard.dev/) for release trust.
- Use [OpenTelemetry](https://opentelemetry.io/docs/concepts/signals/) and the
  [DORA metrics guide](https://dora.dev/guides/dora-metrics/) for outcomes.
- Use [Backstage's catalog model](https://backstage.io/docs/features/software-catalog/)
  as a composition reference, not a replacement for repository governance.

## Portfolio governance

- Reassess priority after each minor release using verified evidence.
- Update target dates when assumptions change; never edit dates to imply actuals.
- Create an ADR before changing a structural or public contract.
- Keep one active roadmap membership per spec.
- Close every implemented spec with validation and follow-up disposition.
- Mark obsolete work `superseded` or `abandoned`; do not delete history silently.
