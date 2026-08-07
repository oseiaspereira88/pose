---
schema_version: 1
generated_at: 2026-08-07T19:25:32Z
baseline_commit: 715b333309ff970299f77edb48af34aa3787eb5a
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
<!-- state:derived hash:cee5957ec20b -->

- specs: total=59 draft=0 in-progress=0 blocked=0 done=59 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-governance-gate-activation (2026-08-07)
  - spec:pose-first-release-evidence-confirmation (2026-08-07)
  - spec:pose-assessment-engine-precision (2026-08-07)
  - spec:pose-release-cycle-debt-closure (2026-08-07)
  - spec:pose-command-reference-parity (2026-08-07)
  - ... e mais 54 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:85cef6ad8a96 -->

- abertos: 33
- por criticidade: high=4 medium=13 low=16 sem-classificação=0
- vencidos (review < hoje): 0

## Capabilities
<!-- state:derived hash:26c78e453c1a -->

- assessment: presente, baseline_commit=commit:c9a08fa, assessed_at=2026-07-22 (2 dias atrás)
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
<!-- state:derived hash:89c8b347eaf5 -->

- último registro: task=validate-native outcome=fail (2026-08-07T19:08:48Z)
- últimos 30 dias: total=109 outcome_ok=94 outcome_outro=15
- reports revisados (.md): total=48
  - report:2026-08-07-standard-validate-native.md
  - report:2026-08-07-standard-closeout-pose-extension-signing-clean-tree.md
  - report:2026-08-07-standard-release-v0-20-1.md
  - report:2026-08-07-standard-release-v0-20-0.md
  - report:2026-08-07-standard-closeout-pose-compat-gate-manual-refresh-assertion.md

## Arquitetura
<!-- state:derived hash:0a483847335f status:active -->

- componentes: total=2 verificados=2 completude=99.0%
- linhas_de_codigo: producao=26964 testes=16567 total=43531
- linguagens: go
- saude_de_codigo: TODOs=0 FIXMEs=0 panics=1 stubs=0
- integracoes: contratos=50 ativos=1 gaps=49
- divida_tecnica: total=1 coberta=0 descoberta=1
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
