---
schema_version: 1
generated_at: 2026-09-25T04:53:25Z
baseline_commit: bce2e6122060f5f96c802468a190e503e57ca09e
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
<!-- state:derived hash:c28ca238c8e9 -->

- specs: total=232 draft=4 in-progress=65 blocked=0 done=163 superseded=0 abandoned=0
- roadmaps: total=12 active=2 done=10
- últimos closeouts:
  - spec:pose-agent-project-context (2026-09-25)
  - spec:pose-spec-authority-transfer (2026-09-24)
  - spec:pose-federated-roadmap-acceptance (2026-09-24)
  - spec:pose-qualified-artifact-resolution (2026-09-21)
  - spec:check-worker-count-is-the-machines (2026-09-21)
  - ... e mais 158 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:c71a4814d371 -->

- abertos: 110
- por criticidade: high=1 medium=25 low=60 sem-classificação=24
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
<!-- state:derived hash:e53a947062a9 -->

- último registro: task=validate-native outcome=pass (2026-09-25T01:16:50Z)
- últimos 30 dias: total=88 outcome_ok=78 outcome_outro=10
- reports revisados (.md): total=145
  - report:2026-09-25-standard-validate-native.md
  - report:2026-09-24-multirepo-autonomous-review.md
  - report:2026-09-24-standard-validate-native.md
  - report:2026-09-21-standard-validate-native.md
  - report:2026-09-21-qualified-artifact-resolution-review.md

## Arquitetura
<!-- state:derived hash:c904d7a3aa6c status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=50725 testes=38694 total=89419
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=57 ativos=1 gaps=56
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
