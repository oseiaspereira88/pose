---
schema_version: 1
generated_at: 2026-09-19T02:26:44Z
baseline_commit: 0bcc21308c00a72b2cc8c859b772946442dba266
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
<!-- state:derived hash:b6fdd7aaea84 -->

- specs: total=211 draft=4 in-progress=66 blocked=0 done=141 superseded=0 abandoned=0
- roadmaps: total=11 active=1 done=10
- últimos closeouts:
  - spec:pose-abm-review-soundness (2026-09-19)
  - spec:pose-discovery-gitignore-inside-submodules (2026-09-15)
  - spec:pose-covered-followup-anchor-verification (2026-08-23)
  - spec:pose-review-bundle-scope-isolated-listing (2026-08-23)
  - spec:pose-delivery-target-root-module-evidence-matching (2026-08-23)
  - ... e mais 136 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:4b1d2ec9297c -->

- abertos: 99
- por criticidade: high=1 medium=17 low=57 sem-classificação=24
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
<!-- state:derived hash:d4881ab567b9 -->

- último registro: task=validate-native outcome=pass (2026-09-19T01:45:01Z)
- últimos 30 dias: total=46 outcome_ok=45 outcome_outro=1
- reports revisados (.md): total=140
  - report:2026-09-19-standard-validate-native.md
  - report:2026-09-15-standard-validate-native.md
  - report:2026-09-12-standard-validate-native.md
  - report:2026-09-11-standard-validate-native.md
  - report:2026-09-10-standard-validate-native.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
