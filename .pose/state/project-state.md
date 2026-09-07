---
schema_version: 1
generated_at: 2026-09-07T04:29:55Z
baseline_commit: 3ed4cc013d0efb689c0e2528116030a0c2abda10
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
<!-- state:derived hash:60912ebd18fd -->

- specs: total=150 draft=6 in-progress=4 blocked=0 done=140 superseded=0 abandoned=0
- roadmaps: total=11 active=1 done=10
- últimos closeouts:
  - spec:pose-public-claims-contract (2026-09-07)
  - spec:pose-review-bundle-scope-isolated-listing (2026-08-23)
  - spec:pose-covered-followup-anchor-verification (2026-08-23)
  - spec:pose-delivery-target-root-module-evidence-matching (2026-08-23)
  - spec:pose-review-bundle-doc-only-specs (2026-08-22)
  - ... e mais 135 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:1c6de17869c5 -->

- abertos: 80
- por criticidade: high=1 medium=20 low=36 sem-classificação=23
- vencidos (review < hoje): 2
  - spec:pose-v1-2-2-changelog-review (owner:@pose-maintainers review:2026-09-01)
  - spec:capability:mcp-agent-interop (owner:unowned review:2026-09-04)

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
<!-- state:derived hash:6b24a7d4c130 -->

- último registro: task=feature outcome=unknown (2026-08-22T05:35:43Z)
- últimos 30 dias: total=123 outcome_ok=80 outcome_outro=43
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
