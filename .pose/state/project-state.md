---
schema_version: 1
generated_at: 2026-08-15T06:24:27Z
baseline_commit: 8537ee8613c82b50eaebfded47230bcd9284a9bc
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
<!-- state:derived hash:3b76843a2bbc -->

- specs: total=98 draft=0 in-progress=0 blocked=0 done=98 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-v1-2-2-changelog-review (2026-08-15)
  - spec:pose-review-bundle-root-file-classification (2026-08-15)
  - spec:pose-review-scope-trailer-check (2026-08-15)
  - spec:pose-scaffold-self-referential-policy-fix (2026-08-15)
  - spec:pose-unified-review-convergence (2026-08-15)
  - ... e mais 93 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:3d5e0a464a07 -->

- abertos: 51
- por criticidade: high=1 medium=16 low=34 sem-classificação=0
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
<!-- state:derived hash:2ccfd8b48815 -->

- último registro: task=closeout-pose-install-gate-failure-recovery-notice outcome=unknown (2026-08-15T05:30:29Z)
- últimos 30 dias: total=229 outcome_ok=164 outcome_outro=65
- reports revisados (.md): total=112
  - report:2026-08-15-standard-closeout-pose-install-gate-failure-recovery-notice.md
  - report:2026-08-15-standard-closeout-pose-install-locale-autodetect.md
  - report:2026-08-15-standard-closeout-pose-v1-2-2-changelog-review-delivery-fix.md
  - report:2026-08-15-standard-closeout-pose-v1-2-2-changelog-review-follow-up-note.md
  - report:2026-08-15-standard-closeout-pose-v1-2-2-changelog-review.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
