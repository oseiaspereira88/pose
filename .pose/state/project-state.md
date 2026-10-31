---
schema_version: 1
generated_at: 2026-10-31T02:11:45Z
baseline_commit: d434349870084928be17e658e5bbb55405aa0ee6
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
<!-- state:derived hash:b14fe9051018 -->

- specs: total=464 draft=20 in-progress=18 blocked=0 done=425 superseded=0 abandoned=1
- roadmaps: total=23 active=3 done=11
- últimos closeouts:
  - spec:pose-cli-test-wall-time (2026-10-31)
  - spec:pose-delegation-result-before-marker (2026-10-31)
  - spec:pose-delegation-parity-gate (2026-10-30)
  - spec:pose-review-subject-later-removal (2026-10-30)
  - spec:pose-validation-matrix-dead-test-patterns (2026-10-30)
  - ... e mais 420 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:509bf930f1a5 -->

- abertos: 187
- por criticidade: high=4 medium=28 low=136 sem-classificação=19
- vencidos (review < hoje): 32
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 22 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:9a109e7387b1 -->

- ADRs: total=90
  - adr:2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
  - adr:2026-10-26-delegation-is-a-first-class-execution-surface.md
  - adr:2026-10-25-reserve-delegated-attempts-before-launch-and-witness-immutable-runs.md
  - adr:2026-10-25-continue-delegated-runs-under-one-immutable-attempt.md
  - adr:2026-10-25-bound-trusted-reviewer-execution-and-disclose-unavailable-isolation.md
- knowledge: total=18 ativo=18 expirado=0

## Validação & Evidência
<!-- state:derived hash:58800a03e390 -->

- último registro: task=validate-native outcome=pass (2026-10-24T18:00:55Z)
- últimos 30 dias: total=52 outcome_ok=33 outcome_outro=19
- reports revisados (.md): total=284
  - report:2026-10-29-launcher-parity-failing-first.md
  - report:delegation-adoption-review-round4.md
  - report:delegation-adoption-review-round3.md
  - report:delegation-adoption-review-round2.md
  - report:delegation-adoption-review-round1.md

## Arquitetura
<!-- state:derived hash:ee9e0d84b942 status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=99973 testes=89234 total=189207
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
