---
schema_version: 1
generated_at: 2026-08-14T00:40:37Z
baseline_commit: fdca3756e183586e3b260edd2c73a85a28a28374
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
<!-- state:derived hash:8646aff3eddc -->

- specs: total=91 draft=0 in-progress=0 blocked=0 done=91 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-review-bundle-convergence (2026-08-14)
  - spec:pose-component-aware-review-plans (2026-08-13)
  - spec:pose-managed-doc-project-identity-regression (2026-08-11)
  - spec:pose-dora-five-metrics-v2 (2026-08-11)
  - spec:pose-usage-metrics (2026-08-11)
  - ... e mais 86 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:003cc85f278e -->

- abertos: 42
- por criticidade: high=1 medium=14 low=27 sem-classificação=0
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
<!-- state:derived hash:854e45c73558 -->

- último registro: task=review-bundle-convergence outcome=pass (2026-08-14T00:23:21Z)
- últimos 30 dias: total=208 outcome_ok=153 outcome_outro=55
- reports revisados (.md): total=98
  - report:2026-08-14-standard-review-bundle-convergence.md
  - report:2026-08-13-standard-review-bundle-convergence.md
  - report:2026-08-13-standard-component-aware-review-provenance.md
  - report:2026-08-13-standard-validate-native.md
  - report:2026-08-11-standard-pose-v1-docs-audit.md

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
