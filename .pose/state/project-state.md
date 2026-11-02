---
schema_version: 1
generated_at: 2026-11-02T00:26:35Z
baseline_commit: ecaafae6c7b5f5718e0d42e89e0f169bbc49b71c
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
<!-- state:derived hash:7253e48ae4c6 -->

- specs: total=476 draft=18 in-progress=18 blocked=0 done=438 superseded=0 abandoned=2
- roadmaps: total=23 active=3 done=13
- últimos closeouts:
  - spec:pose-roadmap-close-keeps-its-approvals (2026-11-01)
  - spec:pose-process-inventory-tolerates-exit (2026-11-01)
  - spec:pose-closeout-names-the-missing-trailer (2026-11-01)
  - spec:pose-reports-name-the-declared-project (2026-11-01)
  - spec:pose-milestone-closes-when-reached (2026-11-01)
  - ... e mais 433 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:ed06c5d43648 -->

- abertos: 206
- por criticidade: high=6 medium=40 low=141 sem-classificação=19
- vencidos (review < hoje): 33
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
  - ... e mais 23 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:a7f583a5a2e6 -->

- ADRs: total=92
  - adr:2026-11-01-a-milestone-answers-to-its-own-edges.md
  - adr:2026-10-31-review-reads-raw-history-and-roots-diff-from-the-empty-tree.md
  - adr:2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
  - adr:2026-10-26-delegation-is-a-first-class-execution-surface.md
  - adr:2026-10-25-reserve-delegated-attempts-before-launch-and-witness-immutable-runs.md
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
<!-- state:derived hash:5d468a91ba5e status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=101470 testes=92781 total=194251
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
