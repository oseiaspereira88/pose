---
schema_version: 1
generated_at: 2026-08-09T22:15:58Z
baseline_commit: 838c2fd3f4a44c87224c9e5c72ec575de116c110
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
<!-- state:derived hash:44c0bae2d5e8 -->

- specs: total=80 draft=0 in-progress=0 blocked=0 done=80 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-container-build-gate (2026-08-09)
  - spec:pose-skill-command-parity (2026-08-08)
  - spec:pose-release-clean-tree-attribution (2026-08-08)
  - spec:pose-action-runtime-currency-gate (2026-08-08)
  - spec:pose-manual-locale-parity (2026-08-08)
  - ... e mais 75 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:25aee9a1e727 -->

- abertos: 38
- por criticidade: high=0 medium=12 low=26 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:665af9c69b7c -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (18 dias atrás)
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
<!-- state:derived hash:4242d30b2d31 -->

- último registro: task=closeout-pose-container-build-gate-draft outcome=unknown (2026-08-09T22:15:27Z)
- últimos 30 dias: total=140 outcome_ok=100 outcome_outro=40
- reports revisados (.md): total=77
  - report:2026-08-09-standard-closeout-pose-container-build-gate-draft.md
  - report:2026-08-09-standard-closeout-pose-container-build-gate.md
  - report:2026-08-08-standard-closeout-pose-dependency-pin-refresh-proven.md
  - report:2026-08-08-standard-closeout-pose-dependency-pin-refresh.md
  - report:2026-08-08-standard-closeout-pose-sbom-negative-coverage.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
