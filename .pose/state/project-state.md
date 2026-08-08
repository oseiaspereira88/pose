---
schema_version: 1
generated_at: 2026-08-08T13:26:00Z
baseline_commit: 5e2fa88dd436410df4521368a756ecf05344f8d3
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
<!-- state:derived hash:3eca1a93f10b -->

- specs: total=77 draft=0 in-progress=0 blocked=0 done=77 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-skill-command-parity (2026-08-08)
  - spec:pose-package-channel-install-repair (2026-08-08)
  - spec:pose-action-runtime-currency-gate (2026-08-08)
  - spec:pose-release-clean-tree-attribution (2026-08-08)
  - spec:pose-manual-locale-parity (2026-08-08)
  - ... e mais 72 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:c77c64e4e758 -->

- abertos: 37
- por criticidade: high=0 medium=13 low=24 sem-classificação=0
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
<!-- state:derived hash:6462bb81a971 -->

- último registro: task=closeout-pose-action-runtime-currency-gate-draft outcome=unknown (2026-08-08T13:25:22Z)
- últimos 30 dias: total=135 outcome_ok=100 outcome_outro=35
- reports revisados (.md): total=72
  - report:2026-08-08-standard-closeout-pose-action-runtime-currency-gate-draft.md
  - report:2026-08-08-standard-closeout-pose-action-runtime-currency-gate.md
  - report:2026-08-08-standard-closeout-pose-skill-command-parity.md
  - report:2026-08-08-standard-closeout-pose-skill-closeout-gate-parity.md
  - report:2026-08-08-standard-closeout-pose-manual-locale-parity.md

## Arquitetura
<!-- state:derived hash:26bc9a5cafff status:unavailable -->

GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade).

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
