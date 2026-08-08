---
schema_version: 1
generated_at: 2026-08-08T02:43:31Z
baseline_commit: 66d5209d5196efabcff8cc8630d868a4d8c7b284
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
<!-- state:derived hash:076e9172db07 -->

- specs: total=69 draft=1 in-progress=0 blocked=0 done=68 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-release-signing-rejection (2026-08-07)
  - spec:pose-actions-node24-bump (2026-08-07)
  - spec:pose-manual-distribution-merge (2026-08-07)
  - spec:pose-assessment-engine-precision (2026-08-07)
  - spec:pose-verifier-assets-variable-fix (2026-08-07)
  - ... e mais 63 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:9078e99efd1c -->

- abertos: 37
- por criticidade: high=2 medium=17 low=18 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:78115c60373b -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (17 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:4709d372b486 -->

- ADRs: total=37
  - adr:2026-08-06-mcp-active-context-authorized-discovery.md
  - adr:2026-08-03-immutable-release-ledger.md
  - adr:2026-08-02-immutable-hierarchical-review-and-closeout-evidence.md
  - adr:2026-08-02-delivery-integrity-graph-and-git-observed-provenance.md
  - adr:2026-07-19-versioned-validation-result-contract.md
- knowledge: total=2 ativo=2 expirado=0

## Validação & Evidência
<!-- state:derived hash:92593c61cf1f -->

- último registro: task=closeout-pose-dependency-digest-pinning outcome=unknown (2026-08-08T02:37:36Z)
- últimos 30 dias: total=123 outcome_ok=100 outcome_outro=23
- reports revisados (.md): total=60
  - report:2026-08-08-standard-closeout-pose-dependency-digest-pinning.md
  - report:2026-08-07-standard-closeout-pose-package-channel-workflow-safety.md
  - report:2026-08-07-standard-closeout-pose-actions-node24-bump.md
  - report:2026-08-07-standard-closeout-pose-package-channel-delivery.md
  - report:2026-08-07-standard-closeout-pose-release-workflow-hardening.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
