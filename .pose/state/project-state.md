---
schema_version: 1
generated_at: 2026-10-01T23:43:32Z
baseline_commit: f41b5aa25d61afa54461abcc8d6c5d44168e6460
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
<!-- state:derived hash:f29f84ce4133 -->

- specs: total=281 draft=2 in-progress=11 blocked=0 done=268 superseded=0 abandoned=0
- roadmaps: total=12 active=1 done=11
- últimos closeouts:
  - spec:pose-cli-live-helper-coverage (2026-10-01)
  - spec:pose-live-policy-reads-through-aliases (2026-10-01)
  - spec:pose-history-quiet-tolerated-verdict (2026-10-01)
  - spec:pose-suite-history-promise-per-step (2026-10-01)
  - spec:pose-governance-residual-diagnostics (2026-10-01)
  - ... e mais 263 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:3240b415e51a -->

- abertos: 123
- por criticidade: high=7 medium=30 low=62 sem-classificação=24
- vencidos (review < hoje): 2
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)

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
<!-- state:derived hash:2bbd50c07f60 -->

- último registro: task=validate-native outcome=pass (2026-10-01T21:24:46Z)
- últimos 30 dias: total=136 outcome_ok=115 outcome_outro=21
- reports revisados (.md): total=155
  - report:2026-10-01-standard-validate-native.md
  - report:pose-v6-2-0-autonomous-work.md
  - report:2026-09-29-standard-record-the-v2-0-2-change-set-of-pose-adoption-stamp-stays-readable.md
  - report:2026-09-28-review-archive-coattribution.md
  - report:2026-09-28-pose-dist-v6-adoption.md

## Arquitetura
<!-- state:derived hash:19a4164022b2 status:active -->

- componentes: total=3 verificados=3 completude=100.0%
- linhas_de_codigo: producao=54413 testes=43811 total=98224
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=58 ativos=1 gaps=57
- divida_tecnica: total=0 coberta=0 descoberta=0
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
