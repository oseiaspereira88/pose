# ABM dependency implementation review — 2026-09-28

## Rules applied during review

Change type: mixed Go feature, security bugfix and distributed instructions.
Workflow: `.pose/workflows/review.md` and `.pose/workflows/bugfix.md`.
Rules: `security` (confined paths, trust, signatures and authority), `backend-go`
(bounded input, context and integration), `documentation-style` (public and
installed instructions), `knowledge-governance` (preserve historical records).
No React, cluster or Cloudflare implementation is part of these source scopes.
Consulted knowledge:module-metadata-discovery-invalidates-review-provenance.

## Inspected implementation and conclusions

- review-soundness: judgment remains pending until a conclusion is recorded;
  orphan findings and unsupported evidence cannot approve. The residual risk
  bypass was reproduced and fixed in its own spec before approval renewal.
- subject-evidence: content identity excludes provider refs and derived-only
  changes; observed, carried-forward and unknown are distinct, without claiming
  that a historical check observed later code. Confined manifests reject escapes.
- review-authority: Ed25519 issuer/key pins, audience, bundle, expiry, principal
  and separate executions are checked; human role requires the configured grant.
  Prefix identity remains declared assurance where verified authority is opt-in.
- design-basis: the narrow R/A/D parser ignores comments and fences, rejects
  foreign references and separates text digest from local evidence resolution.
- structural-delta: bounded before/after observations detect direct material
  contracts and dependencies; unsupported or transitive input remains coverage,
  not a mandatory architectural preference. Raw source and registry URLs are hashed.
- progressive-review: baseline/elevated/critical explains the effective plan;
  experimental overlays remain opt-in. Trivial scopes gain no obligations and
  an overlay cannot lower the human floor. Incomplete metadata is visible.
- governance-outcomes: project-local read-only counts retain missing telemetry
  and denominators. Schema 2 reconciles duplicate/conflicting report identity,
  filters the correct population and exports bounded opaque lineage references.
- remediation-lineage: closed categories and confined references reject orphan
  links, cycles and excess work. Mature unlinked units and censored observations
  are separate; lack of lineage is unknown, not proof of no remediation.
- review-soundness-residuals: `wont-fix` takes the sealed accepted-risk gate;
  negative decisions remain audit records and cannot approve or be laundered by
  reuse. Store, CLI and signed import regressions passed. The Portuguese skill
  was corrected; the embedded distribution now checks pending judgments.
- retrospective-replay: counterfactual contracts are applied only to memory;
  artifact byte snapshots prove no writes. Limits, invalid records, symlinks,
  missing subjects and unknown structure remain explicit. Export has no principal,
  path or source text. The 24 golden cases reuse explicit independent oracles;
  fixtures are not counted as observed dogfood deliveries or external adoption.

## Evidence and limits

Source full matrix: 36/36 passed at `e344a45`, including enforcement module,
all source unit/build/vet checks, contract integrations and CLI reachability.
Compatibility: v6.0.0 candidate passed authenticated upgrades from 5.0.8,
1.1.0, 1.0.0, 0.19.0 and 0.18.2; populated pt-BR instances retained custom
manual content, specs and knowledge through idempotent update. Distribution
and version contracts passed. The 5.0.8 checksum pin matches the provider digest.

Assessments: 58 source contracts, 1 active and 57 unobserved consumer gaps,
zero uncovered source debt markers. These are baseline visibility limitations,
not claims that all possible external consumers are integrated. History and
skill checks passed. Outcomes v2 surface check has no findings; remediation
lineage uses containing-module coverage with warnings. Docs-site has no
registered validation producer and its planned tool is explicitly not-used.
New scope surface and attribution checks are recorded in the structured result.

## Decision and recording

Technical dependency implementations examined above are approved within their
spec scope. Mechanical evidence does not supply these reviewer conclusions.
Current bundle/plan digests, explicit judgment/tool dispositions and post-review
gates are recorded separately by the canonical source CLI. Source vulnerability scan passed both modules with govulncheck 1.4.0;
release readiness remains pending remaining security/publication gates; this report declares no
release published, no human acceptance and no external adoption.
