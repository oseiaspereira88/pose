---
schema_version: 1
generated_at: 2026-09-19T17:17:25Z
baseline_commit: 030f364f62ffd37b9e7ca36fff057b9b92177e48
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
<!-- state:derived hash:cca984d9b57c -->

- specs: total=217 draft=4 in-progress=68 blocked=0 done=145 superseded=0 abandoned=0
- roadmaps: total=11 active=1 done=10
- últimos closeouts:
  - spec:pose-abm-review-soundness (2026-09-19)
  - spec:pose-abm-review-tool-deferred-without-delivery (2026-09-19)
  - spec:pose-abm-structural-delta (2026-09-19)
  - spec:pose-abm-subject-evidence (2026-09-19)
  - spec:pose-abm-authority-fixture-clock (2026-09-19)
  - ... e mais 140 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:79d61cde7e78 -->

- abertos: 102
- por criticidade: high=1 medium=20 low=57 sem-classificação=24
- vencidos (review < hoje): 1
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)

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
<!-- state:derived hash:e2a47ca24261 -->

- último registro: task=validate-native outcome=pass (2026-09-19T16:20:31Z)
- últimos 30 dias: total=68 outcome_ok=62 outcome_outro=6
- reports revisados (.md): total=140
  - report:2026-09-19-standard-validate-native.md
  - report:2026-09-15-standard-validate-native.md
  - report:2026-09-12-standard-validate-native.md
  - report:2026-09-11-standard-validate-native.md
  - report:2026-09-10-standard-validate-native.md

## Arquitetura
<!-- state:derived hash:5593783da5bf status:active -->

- componentes: total=1 verificados=1 completude=100.0%
- linhas_de_codigo: producao=43285 testes=32532 total=75817
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=55 ativos=1 gaps=54
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
