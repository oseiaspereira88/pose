---
schema_version: 1
generated_at: 2026-08-07T04:36:42Z
baseline_commit: fd66001edeb29c10bbdeed6457b90bcc96b391bf
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
<!-- state:derived hash:ae15b01bb54c -->

- specs: total=51 draft=0 in-progress=0 blocked=0 done=51 superseded=0 abandoned=0
- roadmaps: total=8 active=0 done=8
- últimos closeouts:
  - spec:pose-assessment-engine-precision (2026-08-07)
  - spec:pose-manual-distribution-merge (2026-08-07)
  - spec:pose-installer-local-binary-precedence (2026-08-07)
  - spec:pose-command-reference-parity (2026-08-07)
  - spec:pose-compat-gate-candidate-integrity (2026-08-07)
  - ... e mais 46 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:32d527283bad -->

- abertos: 39
- por criticidade: high=5 medium=13 low=16 sem-classificação=5
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
<!-- state:derived hash:0508593bec4e -->

- último registro: task=release-v0-18-1 outcome=pass (2026-08-07T04:23:39Z)
- últimos 30 dias: total=79 outcome_ok=70 outcome_outro=9
- reports revisados (.md): total=36
  - report:2026-08-07-standard-release-v0-18-1.md
  - report:2026-08-07-standard-validate-native.md
  - report:2026-08-07-standard-closeout-pose-installer-local-binary-precedence.md
  - report:2026-08-07-standard-release-v0-18-0.md
  - report:2026-08-07-standard-closeout-pose-release-lifecycle-closure.md

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
