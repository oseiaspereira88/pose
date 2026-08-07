---
schema_version: 1
generated_at: 2026-08-07T19:04:53Z
baseline_commit: 63903e607d60c82380d1194c009e5bc47ce48fc3
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
<!-- state:derived hash:cc34a0613ae1 -->

- specs: total=57 draft=0 in-progress=0 blocked=0 done=57 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-governance-gate-activation (2026-08-07)
  - spec:pose-release-cycle-debt-closure (2026-08-07)
  - spec:pose-assessment-engine-precision (2026-08-07)
  - spec:pose-extension-reference-publication (2026-08-07)
  - spec:pose-manual-distribution-merge (2026-08-07)
  - ... e mais 52 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:4dbf2e3b8031 -->

- abertos: 30
- por criticidade: high=4 medium=10 low=16 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:26c78e453c1a -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (2 dias atrás)
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
<!-- state:derived hash:ba0a02e7697e -->

- último registro: task=release-v0-20-0 outcome=pass (2026-08-07T19:00:38Z)
- últimos 30 dias: total=105 outcome_ok=91 outcome_outro=14
- reports revisados (.md): total=46
  - report:2026-08-07-standard-release-v0-20-0.md
  - report:2026-08-07-standard-validate-native.md
  - report:2026-08-07-standard-closeout-pose-compat-gate-manual-refresh-assertion.md
  - report:2026-08-07-standard-closeout-pose-extension-reference-publication.md
  - report:2026-08-07-standard-release-v0-19-0.md

## Arquitetura
<!-- state:derived hash:0a483847335f status:active -->

- componentes: total=2 verificados=2 completude=99.0%
- linhas_de_codigo: producao=26964 testes=16567 total=43531
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=1 stubs=0
- integracoes: contratos=50 ativos=1 gaps=49
- divida_tecnica: total=1 coberta=0 descoberta=1
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
