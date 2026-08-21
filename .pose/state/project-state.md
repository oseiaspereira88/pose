---
schema_version: 1
generated_at: 2026-08-21T16:44:53Z
baseline_commit: 9cca078cbd00c11ad5c817b5b025931f9ab2f673
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
<!-- state:derived hash:a98ce13fe3ff -->

- specs: total=120 draft=1 in-progress=0 blocked=0 done=119 superseded=0 abandoned=0
- roadmaps: total=10 active=0 done=10
- últimos closeouts:
  - spec:pose-cli-ergonomics-and-stack-expansion (2026-08-21)
  - spec:pose-engine-stability-and-diagnostics-convergence (2026-08-21)
  - spec:pose-spec-trailer-workflow-documentation (2026-08-21)
  - spec:pose-closeout-delivery-assurance-convergence (2026-08-21)
  - spec:pose-discovery-gitignore-and-root-alias-fix (2026-08-17)
  - ... e mais 114 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:59c7c2a6bb92 -->

- abertos: 74
- por criticidade: high=1 medium=19 low=36 sem-classificação=18
- vencidos (review < hoje): 0

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
<!-- state:derived hash:5656f5e8f783 -->

- último registro: task=validate-native outcome=pass (2026-08-21T06:32:23Z)
- últimos 30 dias: total=223 outcome_ok=154 outcome_outro=69
- reports revisados (.md): total=132
  - report:2026-08-21-standard-validate-native.md
  - report:2026-08-18-standard-validate-native.md
  - report:2026-08-17-standard-closeout-pose-discovery-gitignore-and-root-alias-fix.md
  - report:2026-08-16-standard-closeout-pose-update-instance-directory-completeness.md
  - report:2026-08-16-standard-closeout-pose-derived-index-self-referential-leak.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
