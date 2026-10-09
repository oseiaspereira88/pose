---
schema_version: 1
generated_at: 2026-10-09T15:24:14Z
baseline_commit: 2c4d56959c66c71452614521dfbaa44b30ee04bb
staleness_policy: max_age_days=7,max_commits=20
refresh_pending: 
---

# Project State

## Resumo executivo
<!-- state:curated -->

`pose-dist` é a distribuição standalone do POSE (Project Operating Standard for
Engineering): o binário Go nativo (`pose`), servidor MCP, docs-site e todo o
material publicável do produto POSE em si — trabalha em paralelo ao repositório
`harne8`, que consome esta distribuição via `pose-dist/pose-mcp` como
dependência do próprio ecossistema.

## Direção atual
<!-- state:curated -->

Trabalho corrente puxado pelo repositório `harne8` (roadmap
`harne8-semantic-state-governance`): este ciclo adicionou o artefato nativo de
`project-state` (`pose state`/`pose_project_state`) ao motor Go. Backlog nativo
próprio deste repositório (produtização v2, DX) segue disponível conforme
capacidade.

## Specs & Roadmaps
<!-- state:derived hash:d8c7868eba52 -->

- specs: total=359 draft=1 in-progress=0 blocked=0 done=358 superseded=0 abandoned=0
- roadmaps: total=17 active=1 done=11
- últimos closeouts:
  - spec:pose-red-signal-clears-only-at-the-head (2026-10-09)
  - spec:pose-docs-canonical-route (2026-10-09)
  - spec:pose-locale-obligation-flags (2026-10-09)
  - spec:pose-upgrade-journey-starts-below-the-newest-capability (2026-10-09)
  - spec:pose-trace-test-refs-resolve (2026-10-09)
  - ... e mais 353 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:ad1a012c750b -->

- abertos: 108
- por criticidade: high=4 medium=23 low=62 sem-classificação=19
- vencidos (review < hoje): 10
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)

## Capabilities
<!-- state:derived hash:7db5fb52757a -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (0 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:f78c15dbd473 -->

- ADRs: total=63
  - adr:2026-10-04-obligations-are-projected-action-requests-are-persisted.md
  - adr:2026-10-04-mcp-governance-write-risk-class.md
  - adr:2026-10-02-trusted-dependabot-runtime-repairs.md
  - adr:2026-10-02-release-version-source-is-declared-by-the-project.md
  - adr:2026-10-01-scoped-git-batch-reader-for-structural-assessment.md
- knowledge: total=7 ativo=7 expirado=0

## Validação & Evidência
<!-- state:derived hash:ec59bdfafda6 -->

- último registro: task=validate-native outcome=pass (2026-10-02T11:43:40Z)
- últimos 30 dias: total=139 outcome_ok=117 outcome_outro=22
- reports revisados (.md): total=166
  - report:2026-10-pose-open-backlog-reconciliation.md
  - report:pose-agency-readiness-pilot.md
  - report:pose-abm-capability-adoption.md
  - report:pose-v6-2-0-autonomous-work.md
  - report:pose-v7-legacy-cleanup-plan.md

## Arquitetura
<!-- state:derived hash:19a4164022b2 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=54413 testes=43811 total=98224
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=58 ativos=1 gaps=57
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
