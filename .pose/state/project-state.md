---
schema_version: 1
generated_at: 2026-08-15T12:34:08Z
baseline_commit: a1fddade482b7020840d4e8d59e8779a3abcd114
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
<!-- state:derived hash:c11b12b4f101 -->

- specs: total=105 draft=0 in-progress=0 blocked=0 done=105 superseded=0 abandoned=0
- roadmaps: total=9 active=0 done=8
- últimos closeouts:
  - spec:pose-stack-detection-consolidation (2026-08-15)
  - spec:pose-scaffold-index-template-neutralization (2026-08-15)
  - spec:pose-adaptive-rule-delivery (2026-08-15)
  - spec:pose-install-gate-failure-recovery-notice (2026-08-15)
  - spec:pose-instance-engine-version-tracking (2026-08-15)
  - ... e mais 100 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:14c9c3e3b9f4 -->

- abertos: 59
- por criticidade: high=1 medium=16 low=34 sem-classificação=8
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
<!-- state:derived hash:d51852bc3d1c -->

- último registro: task=advise-on-redundant-monorepo-execution-add-root-only-workspace-flags outcome=pass (2026-08-15T12:33:13Z)
- últimos 30 dias: total=237 outcome_ok=171 outcome_outro=66
- reports revisados (.md): total=120
  - report:2026-08-15-standard-advise-on-redundant-monorepo-execution-add-root-only-workspace-flags.md
  - report:2026-08-15-standard-recommend-rule-extension-matching-detected-stack.md
  - report:2026-08-15-standard-seed-module-metadata-json-from-discovered-modules.md
  - report:2026-08-15-standard-fix-review-bundle-gaps-surfaced-by-rule-extension-migration.md
  - report:2026-08-15-standard-migrate-backend-go-frontend-react-rules-to-extensions.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
