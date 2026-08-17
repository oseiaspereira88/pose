---
schema_version: 1
generated_at: 2026-08-17T01:10:36Z
baseline_commit: f127f7db626b0c2ca23cb54f4f1a6d4bd838325f
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
<!-- state:derived hash:9f57a9f6d4a1 -->

- specs: total=115 draft=0 in-progress=0 blocked=0 done=115 superseded=0 abandoned=0
- roadmaps: total=10 active=0 done=10
- últimos closeouts:
  - spec:pose-discovery-gitignore-and-root-alias-fix (2026-08-17)
  - spec:pose-derived-index-self-referential-leak (2026-08-16)
  - spec:pose-update-instance-directory-completeness (2026-08-16)
  - spec:pose-upgrade-path-audit-fixes (2026-08-16)
  - spec:pose-scaffold-index-template-neutralization (2026-08-15)
  - ... e mais 110 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:9bb935aad49e -->

- abertos: 72
- por criticidade: high=1 medium=18 low=36 sem-classificação=17
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:d98c218934f8 -->

- assessment: presente, baseline_commit=commit:b65156e, assessed_at=2026-08-13 (4 dias atrás)
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
<!-- state:derived hash:f236392abd12 -->

- último registro: task=closeout-pose-discovery-gitignore-and-root-alias-fix outcome=unknown (2026-08-17T00:45:47Z)
- últimos 30 dias: total=247 outcome_ok=177 outcome_outro=70
- reports revisados (.md): total=130
  - report:2026-08-17-standard-closeout-pose-discovery-gitignore-and-root-alias-fix.md
  - report:2026-08-16-standard-closeout-pose-update-instance-directory-completeness.md
  - report:2026-08-16-standard-closeout-pose-derived-index-self-referential-leak.md
  - report:2026-08-16-standard-closeout-pose-upgrade-path-audit-fixes.md
  - report:2026-08-15-standard-resolve-an-extension-id-against-the-latest-published-release.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
