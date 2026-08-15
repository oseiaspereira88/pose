---
schema_version: 1
generated_at: 2026-08-15T11:49:53Z
baseline_commit: 4292ee64c7ba85034051aff03ded5db489a983ea
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
<!-- state:derived hash:b58de03095da -->

- specs: total=105 draft=4 in-progress=0 blocked=0 done=101 superseded=0 abandoned=0
- roadmaps: total=9 active=0 done=8
- últimos closeouts:
  - spec:pose-review-scope-trailer-check (2026-08-15)
  - spec:pose-install-locale-autodetect (2026-08-15)
  - spec:pose-v1-2-2-changelog-review (2026-08-15)
  - spec:pose-instance-engine-version-tracking (2026-08-15)
  - spec:pose-unified-review-convergence (2026-08-15)
  - ... e mais 96 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:82b91ea0c04d -->

- abertos: 56
- por criticidade: high=1 medium=16 low=34 sem-classificação=5
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:f919050f8b00 -->

- assessment: presente, baseline_commit=commit:b65156e, assessed_at=2026-08-13 (2 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:cb5dc576eaf0 -->

- ADRs: total=43
  - adr:2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md
  - adr:2026-08-14-unified-review-convergence-and-auto-attestation.md
  - adr:2026-08-13-sealed-review-bundles-and-attestations.md
  - adr:2026-08-12-component-aware-effective-review-plans.md
  - adr:2026-08-10-production-scoped-dora-five-metric-contract.md
- knowledge: total=9 ativo=9 expirado=0

## Validação & Evidência
<!-- state:derived hash:ba315db97e56 -->

- último registro: task=neutralize-leaked-pose-mcp-index-templates outcome=pass (2026-08-15T11:47:42Z)
- últimos 30 dias: total=232 outcome_ok=166 outcome_outro=66
- reports revisados (.md): total=115
  - report:2026-08-15-standard-neutralize-leaked-pose-mcp-index-templates.md
  - report:2026-08-15-standard-resolve-durable-non-architectural-knowledge-routing-adr-rules-skill-guidance.md
  - report:2026-08-15-standard-closeout-pose-instance-engine-version-tracking.md
  - report:2026-08-15-standard-closeout-pose-install-gate-failure-recovery-notice.md
  - report:2026-08-15-standard-closeout-pose-install-locale-autodetect.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
