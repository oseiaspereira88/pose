---
schema_version: 1
generated_at: 2026-08-16T02:37:51Z
baseline_commit: 62685de162a2ed5a3e77f8951fad124b6fc7532e
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
<!-- state:derived hash:cec23be6163a -->

- specs: total=111 draft=0 in-progress=0 blocked=0 done=111 superseded=0 abandoned=0
- roadmaps: total=10 active=0 done=10
- últimos closeouts:
  - spec:pose-domain-rule-extension-migration (2026-08-15)
  - spec:pose-review-scope-trailer-check (2026-08-15)
  - spec:pose-adaptive-rule-delivery (2026-08-15)
  - spec:pose-v1-2-2-changelog-review (2026-08-15)
  - spec:pose-install-gate-failure-recovery-notice (2026-08-15)
  - ... e mais 106 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:ad18e7ac3a5c -->

- abertos: 68
- por criticidade: high=1 medium=16 low=34 sem-classificação=17
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:6245873593fe -->

- assessment: presente, baseline_commit=commit:b65156e, assessed_at=2026-08-13 (3 dias atrás)
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
<!-- state:derived hash:588caba2082d -->

- último registro: task=resolve-an-extension-id-against-the-latest-published-release outcome=pass (2026-08-15T19:45:23Z)
- últimos 30 dias: total=243 outcome_ok=177 outcome_outro=66
- reports revisados (.md): total=126
  - report:2026-08-15-standard-resolve-an-extension-id-against-the-latest-published-release.md
  - report:2026-08-15-standard-support-a-locale-overlay-in-pose-extension-install.md
  - report:2026-08-15-standard-consolidate-module-stack-detection-into-one-shared-function.md
  - report:2026-08-15-standard-excerpt-readme-claude-md-into-agents-md-s-project-context.md
  - report:2026-08-15-standard-fix-test-self-exec-flake-blocking-v1-3-0.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
