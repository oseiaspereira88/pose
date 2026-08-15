---
schema_version: 1
generated_at: 2026-08-15T00:07:49Z
baseline_commit: 06c5fa159c1e2c2f031bfb41f8f8dc0b09d0a244
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
<!-- state:derived hash:d67f8afd1e3b -->

- specs: total=93 draft=0 in-progress=1 blocked=0 done=92 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-scaffold-self-referential-policy-fix (2026-08-15)
  - spec:pose-review-bundle-convergence (2026-08-14)
  - spec:pose-component-aware-review-plans (2026-08-13)
  - spec:pose-dora-five-metrics-v2 (2026-08-11)
  - spec:pose-usage-metrics (2026-08-11)
  - ... e mais 87 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:86e1c574b657 -->

- abertos: 44
- por criticidade: high=1 medium=15 low=28 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:700fe5db143b -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (19 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:4709d372b486 -->

- ADRs: total=37
  - adr:2026-08-06-mcp-active-context-authorized-discovery.md
  - adr:2026-08-03-immutable-release-ledger.md
  - adr:2026-08-02-immutable-hierarchical-review-and-closeout-evidence.md
  - adr:2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - adr:2026-07-19-versioned-validation-result-contract.md
- knowledge: total=2 ativo=2 expirado=0

## Validação & Evidência
<!-- state:derived hash:2dcd9a9be604 -->

- último registro: task=validate-native outcome=pass (2026-08-15T00:03:21Z)
- últimos 30 dias: total=217 outcome_ok=162 outcome_outro=55
- reports revisados (.md): total=103
  - report:2026-08-15-standard-validate-native.md
  - report:2026-08-14-standard-review-bundle-convergence.md
  - report:2026-08-14-standard-validate-native.md
  - report:2026-08-14-standard-review-bundle-convergence-release-attribution.md
  - report:2026-08-14-standard-component-aware-review-release-attribution.md

## Arquitetura
<!-- state:derived hash:d03cb9bcadab status:active -->

- componentes: total=2 verificados=2 completude=99.0%
- linhas_de_codigo: producao=32010 testes=19943 total=51953
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=1 stubs=0
- integracoes: contratos=53 ativos=1 gaps=52
- divida_tecnica: total=1 coberta=1 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
