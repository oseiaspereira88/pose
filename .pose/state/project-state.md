---
schema_version: 1
generated_at: 2026-08-15T12:18:32Z
baseline_commit: c7d418b2a2075e1d72c3a04a31450332cff27bd7
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
<!-- state:derived hash:689ed6133977 -->

- specs: total=105 draft=2 in-progress=0 blocked=0 done=103 superseded=0 abandoned=0
- roadmaps: total=9 active=0 done=8
- últimos closeouts:
  - spec:pose-install-gate-failure-recovery-notice (2026-08-15)
  - spec:pose-v1-2-2-changelog-review (2026-08-15)
  - spec:pose-review-bundle-root-file-classification (2026-08-15)
  - spec:pose-review-scope-trailer-check (2026-08-15)
  - spec:pose-domain-rule-extension-migration (2026-08-15)
  - ... e mais 98 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:2b02d03b9d42 -->

- abertos: 58
- por criticidade: high=1 medium=16 low=34 sem-classificação=7
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:f919050f8b00 -->

- assessment: presente, baseline_commit=commit:b65156e, assessed_at=2026-08-13 (2 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:440f0b570fba -->

- ADRs: total=44
  - adr:2026-08-15-retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update.md
  - adr:2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md
  - adr:2026-08-14-unified-review-convergence-and-auto-attestation.md
  - adr:2026-08-13-sealed-review-bundles-and-attestations.md
  - adr:2026-08-12-component-aware-effective-review-plans.md
- knowledge: total=9 ativo=9 expirado=0

## Validação & Evidência
<!-- state:derived hash:992f08d7ade0 -->

- último registro: task=seed-module-metadata-json-from-discovered-modules outcome=pass (2026-08-15T12:17:40Z)
- últimos 30 dias: total=235 outcome_ok=169 outcome_outro=66
- reports revisados (.md): total=118
  - report:2026-08-15-standard-seed-module-metadata-json-from-discovered-modules.md
  - report:2026-08-15-standard-fix-review-bundle-gaps-surfaced-by-rule-extension-migration.md
  - report:2026-08-15-standard-migrate-backend-go-frontend-react-rules-to-extensions.md
  - report:2026-08-15-standard-neutralize-leaked-pose-mcp-index-templates.md
  - report:2026-08-15-standard-resolve-durable-non-architectural-knowledge-routing-adr-rules-skill-guidance.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
