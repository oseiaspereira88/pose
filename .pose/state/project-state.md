---
schema_version: 1
generated_at: 2026-08-11T01:13:35Z
baseline_commit: e110a5d9334111e6221d9a8385a9681186684cdf
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
<!-- state:derived hash:a443296c9e67 -->

- specs: total=89 draft=0 in-progress=0 blocked=0 done=89 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-managed-doc-project-identity-regression (2026-08-11)
  - spec:pose-dora-five-metrics-v2 (2026-08-11)
  - spec:pose-usage-metrics (2026-08-11)
  - spec:pose-scaffold-exclusion-policy (2026-08-10)
  - spec:pose-fragment-error-clarity (2026-08-10)
  - ... e mais 84 (ver `pose_list_specs status:done`)

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
<!-- state:derived hash:3a5ea7f0c0c3 -->

- último registro: task=pose-analytics-final-closeout outcome=pass (2026-08-11T01:03:21Z)
- últimos 30 dias: total=164 outcome_ok=114 outcome_outro=50
- reports revisados (.md): total=93
  - report:2026-08-11-standard-pose-analytics-final-closeout.md
  - report:2026-08-11-standard-pose-analytics-delivery-targets.md
  - report:2026-08-11-standard-pose-dora-five-metrics-v2.md
  - report:2026-08-11-standard-pose-managed-doc-project-identity-regression.md
  - report:2026-08-11-standard-pose-usage-metrics.md

## Arquitetura
<!-- state:derived hash:cb25d685317d status:active -->

- componentes: total=1 verificados=1 completude=98.0%
- linhas_de_codigo: producao=27505 testes=17372 total=44877
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=1 stubs=0
- integracoes: contratos=51 ativos=1 gaps=50
- divida_tecnica: total=1 coberta=0 descoberta=1
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
