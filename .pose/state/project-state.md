---
schema_version: 1
generated_at: 2026-08-15T02:24:58Z
baseline_commit: 26519852c53495ece3fab7ed594f7a72bce54412
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
<!-- state:derived hash:944c20256761 -->

- specs: total=94 draft=0 in-progress=0 blocked=0 done=94 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-unified-review-convergence (2026-08-15)
  - spec:pose-review-bundle-root-file-classification (2026-08-15)
  - spec:pose-scaffold-self-referential-policy-fix (2026-08-15)
  - spec:pose-review-bundle-convergence (2026-08-14)
  - spec:pose-component-aware-review-plans (2026-08-13)
  - ... e mais 89 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:212f868e5daf -->

- abertos: 45
- por criticidade: high=1 medium=15 low=29 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:f919050f8b00 -->

- assessment: presente, baseline_commit=commit:b65156e, assessed_at=2026-08-13 (2 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:bfdee79251ca -->

- ADRs: total=42
  - adr:2026-08-14-unified-review-convergence-and-auto-attestation.md
  - adr:2026-08-13-sealed-review-bundles-and-attestations.md
  - adr:2026-08-12-component-aware-effective-review-plans.md
  - adr:2026-08-10-production-scoped-dora-five-metric-contract.md
  - adr:2026-08-10-local-usage-events-and-outcome-aware-aggregation.md
- knowledge: total=8 ativo=8 expirado=0

## Validação & Evidência
<!-- state:derived hash:66748661b372 -->

- último registro: task=closeout-pose-review-bundle-root-file-classification outcome=unknown (2026-08-15T02:19:53Z)
- últimos 30 dias: total=222 outcome_ok=164 outcome_outro=58
- reports revisados (.md): total=106
  - report:2026-08-15-standard-closeout-pose-review-bundle-root-file-classification.md
  - report:2026-08-15-standard-closeout-pose-unified-review-convergence.md
  - report:2026-08-15-standard-validate-native.md
  - report:2026-08-15-standard-closeout-pose-scaffold-self-referential-policy-fix.md
  - report:2026-08-14-standard-review-bundle-convergence.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
