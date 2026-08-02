---
schema_version: 1
generated_at: 2026-08-02T17:07:36Z
baseline_commit: 3eaa7db815cff2d50e49c30cfc53c29afd187eba
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
<!-- state:derived hash:88d9737acd92 -->

- specs: total=40 draft=3 in-progress=0 blocked=0 done=37 superseded=0 abandoned=0
- roadmaps: total=8 active=1 done=7
- últimos closeouts:
  - spec:pose-hierarchical-review-closeout (2026-08-02)
  - spec:pose-capability-mechanism (2026-07-21)
  - spec:pose-ossf-security-baseline (2026-07-19)
  - spec:pose-otel-observability (2026-07-19)
  - spec:pose-cross-repo-portfolio (2026-07-19)
  - ... e mais 32 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:7af64d34d2f1 -->

- abertos: 33
- por criticidade: high=5 medium=12 low=15 sem-classificação=1
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:26c78e453c1a -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (2 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:96e441203205 -->

- ADRs: total=33
  - adr:2026-07-19-versioned-validation-result-contract.md
  - adr:2026-07-19-verified-public-install-contract.md
  - adr:2026-07-19-validation-runtime-guardrails-and-harness-delegation.md
  - adr:2026-07-19-upgrade-compatibility-lab-populated-fixtures.md
  - adr:2026-07-19-slsa-build-l2-provenance-claim.md
- knowledge: total=1 ativo=1 expirado=0

## Validação & Evidência
<!-- state:derived hash:bb0d43541813 -->

- último registro: task=validate-native outcome=pass (2026-08-02T15:14:05Z)
- últimos 30 dias: total=27 outcome_ok=26 outcome_outro=1
- reports revisados (.md): total=5
  - report:2026-08-02-pose-hierarchical-review-closeout-review.md
  - report:2026-08-02-standard-validate-native.md
  - report:2026-07-19-standard-validate-native.md
  - report:2026-07-18-doc-audit-product-roadmap-portfolio.md
  - report:README.md

## Arquitetura
<!-- state:derived hash:a8f9c10d3e21 status:active -->

- componentes: total=2 verificados=2 completude=100%
- linhas_de_codigo: producao=27441 testes=14955 total=42396
- linguagens: Rust (ast-engine), Go (graph-core, mcp-server, indexers, collector, enricher, planner, cli, pkg), TypeScript/React (web), WASM
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/components/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
