---
schema_version: 1
generated_at: 2026-08-22T05:44:57Z
baseline_commit: e77faa6cc2a6a72da846feffec0b761ad2816ad2
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
<!-- state:derived hash:f1529bf095f7 -->

- specs: total=128 draft=1 in-progress=0 blocked=0 done=127 superseded=0 abandoned=0
- roadmaps: total=10 active=0 done=10
- últimos closeouts:
  - spec:pose-contributor-mode-workflow-and-cli-hints (2026-08-22)
  - spec:pose-spec-trailer-workflow-documentation (2026-08-21)
  - spec:pose-stack-rule-extensions-expansion (2026-08-21)
  - spec:pose-spec-format-migration-command (2026-08-21)
  - spec:pose-engine-stability-and-diagnostics-convergence (2026-08-21)
  - ... e mais 122 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:48a16c7aa529 -->

- abertos: 75
- por criticidade: high=1 medium=20 low=36 sem-classificação=18
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
<!-- state:derived hash:6f17695e37f9 -->

- último registro: task=feature outcome=unknown (2026-08-22T05:35:43Z)
- últimos 30 dias: total=225 outcome_ok=155 outcome_outro=70
- reports revisados (.md): total=134
  - report:2026-08-22-standard-feature.md
  - report:2026-08-22-standard-validate-native.md
  - report:2026-08-21-standard-validate-native.md
  - report:2026-08-18-standard-validate-native.md
  - report:2026-08-17-standard-closeout-pose-discovery-gitignore-and-root-alias-fix.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
