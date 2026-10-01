---
schema_version: 1
generated_at: 2026-10-01T06:21:18Z
baseline_commit: e92891af0cdb68c9843e9cbfefef967e454e80e0
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
<!-- state:derived hash:815c4c9bec4c -->

- specs: total=272 draft=1 in-progress=11 blocked=0 done=260 superseded=0 abandoned=0
- roadmaps: total=12 active=1 done=11
- últimos closeouts:
  - spec:pose-usage-findings-adjudication (2026-10-01)
  - spec:pose-dist-adopt-published-v6-1-0 (2026-09-30)
  - spec:pose-v6-1-0-release-readiness (2026-09-30)
  - spec:pose-help-names-flags-the-parser-accepts (2026-09-29)
  - spec:pose-compat-gate-pose-md-preservation (2026-09-29)
  - ... e mais 255 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:ba1456fab0ee -->

- abertos: 123
- por criticidade: high=8 medium=29 low=62 sem-classificação=24
- vencidos (review < hoje): 2
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)

## Capabilities
<!-- state:derived hash:7db5fb52757a -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (0 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:b8f47d8fd574 -->

- ADRs: total=44
  - adr:2026-08-15-retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update.md
  - adr:2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md
  - adr:2026-08-14-unified-review-convergence-and-auto-attestation.md
  - adr:2026-08-13-sealed-review-bundles-and-attestations.md
  - adr:2026-08-12-component-aware-effective-review-plans.md
- knowledge: total=10 ativo=10 expirado=0

## Validação & Evidência
<!-- state:derived hash:228a4bfe78ca -->

- último registro: task=record-the-v2-0-2-change-set-of-pose-adoption-stamp-stays-readable outcome=skipped (2026-09-29T19:45:14Z)
- últimos 30 dias: total=128 outcome_ok=111 outcome_outro=17
- reports revisados (.md): total=153
  - report:2026-09-29-standard-record-the-v2-0-2-change-set-of-pose-adoption-stamp-stays-readable.md
  - report:2026-09-28-review-archive-coattribution.md
  - report:2026-09-28-pose-dist-v6-adoption.md
  - report:2026-09-28-abm-dependency-review.md
  - report:2026-09-28-standard-validate-native.md

## Arquitetura
<!-- state:derived hash:690a8f0eaa52 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=54212 testes=43085 total=97297
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=58 ativos=1 gaps=57
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
