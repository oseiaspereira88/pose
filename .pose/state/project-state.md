---
schema_version: 1
generated_at: 2026-08-15T01:45:44Z
baseline_commit: 1f89d44f414c5210fb5e1ea604c7fb26c5538d84
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
<!-- state:derived hash:d67f8afd1e3b -->

- specs: total=93 draft=0 in-progress=1 blocked=0 done=92 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-scaffold-self-referential-policy-fix (2026-08-15)
  - spec:pose-review-bundle-convergence (2026-08-14)
  - spec:pose-component-aware-review-plans (2026-08-13)
  - spec:pose-dora-five-metrics-v2 (2026-08-11)
  - spec:pose-usage-metrics (2026-08-11)
  - ... e mais 87 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:86e1c574b657 -->

- abertos: 44
- por criticidade: high=1 medium=15 low=28 sem-classificação=0
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
<!-- state:derived hash:c46c598d6ebb -->

- último registro: task=validate-native outcome=pass (2026-08-15T00:08:38Z)
- últimos 30 dias: total=220 outcome_ok=164 outcome_outro=56
- reports revisados (.md): total=104
  - report:2026-08-15-standard-validate-native.md
  - report:2026-08-15-standard-closeout-pose-scaffold-self-referential-policy-fix.md
  - report:2026-08-14-standard-review-bundle-convergence.md
  - report:2026-08-14-standard-validate-native.md
  - report:2026-08-14-standard-review-bundle-convergence-release-attribution.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
