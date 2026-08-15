---
schema_version: 1
generated_at: 2026-08-15T19:36:51Z
baseline_commit: a3b19bba59e614f27fc5db3ee98861a429e339e7
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
<!-- state:derived hash:cfa1f39e863d -->

- specs: total=111 draft=1 in-progress=0 blocked=0 done=110 superseded=0 abandoned=0
- roadmaps: total=10 active=1 done=9
- últimos closeouts:
  - spec:pose-install-gate-failure-recovery-notice (2026-08-15)
  - spec:pose-scaffold-index-template-neutralization (2026-08-15)
  - spec:pose-adaptive-rule-delivery (2026-08-15)
  - spec:pose-validation-scanner-consolidation (2026-08-15)
  - spec:pose-v1-2-2-changelog-review (2026-08-15)
  - ... e mais 105 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:ac59c88b7381 -->

- abertos: 66
- por criticidade: high=1 medium=16 low=34 sem-classificação=15
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
<!-- state:derived hash:b7291001cdac -->

- último registro: task=support-a-locale-overlay-in-pose-extension-install outcome=pass (2026-08-15T19:36:03Z)
- últimos 30 dias: total=242 outcome_ok=176 outcome_outro=66
- reports revisados (.md): total=125
  - report:2026-08-15-standard-support-a-locale-overlay-in-pose-extension-install.md
  - report:2026-08-15-standard-consolidate-module-stack-detection-into-one-shared-function.md
  - report:2026-08-15-standard-excerpt-readme-claude-md-into-agents-md-s-project-context.md
  - report:2026-08-15-standard-fix-test-self-exec-flake-blocking-v1-3-0.md
  - report:2026-08-15-standard-fix-roadmap-review-bundle-absolute-path-portability-bug.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
