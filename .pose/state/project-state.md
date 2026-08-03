---
schema_version: 1
generated_at: 2026-08-03T15:49:09Z
baseline_commit: 275188124a7f8d299b155eee56e7f919f958ae24
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
<!-- state:derived hash:8d0e91b6fc07 -->

- specs: total=42 draft=0 in-progress=0 blocked=0 done=42 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-delivery-surface-assurance (2026-08-03)
  - spec:pose-artifact-provenance-ledger (2026-08-03)
  - spec:pose-release-lifecycle-closure (2026-08-03)
  - spec:pose-release-evidence-trigger-fix (2026-08-03)
  - spec:pose-project-agnostic-assessment-engines (2026-08-03)
  - ... e mais 37 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:7af64d34d2f1 -->

- abertos: 33
- por criticidade: high=5 medium=12 low=15 sem-classificação=1
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:26c78e453c1a -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (2 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:96e441203205 -->

- ADRs: total=33
  - adr:2026-07-19-versioned-validation-result-contract.md
  - adr:2026-07-19-verified-public-install-contract.md
  - adr:2026-07-19-validation-runtime-guardrails-and-harness-delegation.md
  - adr:2026-07-19-upgrade-compatibility-lab-populated-fixtures.md
  - adr:2026-07-19-slsa-build-l2-provenance-claim.md
- knowledge: total=1 ativo=1 expirado=0

## Validação & Evidência
<!-- state:derived hash:3bb7cec3c78c -->

- último registro: task=release-trigger-fix-validation outcome=pass (2026-08-03T14:29:22Z)
- últimos 30 dias: total=49 outcome_ok=46 outcome_outro=3
- reports revisados (.md): total=23
  - report:2026-08-03-pose-project-agnostic-assessment-engines-review.md
  - report:2026-08-03-standard-release-trigger-fix-validation.md
  - report:2026-08-03-standard-release-evidence-trigger-fix.md
  - report:2026-08-03-release-evidence-trigger-fix-review.md
  - report:2026-08-03-delivery-integrity-roadmap-review.md

## Arquitetura
<!-- state:derived hash:43d6af1ce90c status:active -->

- componentes: total=2 verificados=2 completude=99.0%
- linhas_de_codigo: producao=26198 testes=16050 total=42248
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=1 stubs=0
- integracoes: contratos=49 ativos=1 gaps=48
- divida_tecnica: total=1 coberta=0 descoberta=1
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
