---
schema_version: 1
generated_at: 2026-10-21T04:12:09Z
baseline_commit: d6ce7bc83daec17e8f760580450d14472bb77148
staleness_policy: max_age_days=7,max_commits=20
refresh_pending: spec_closeout
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
<!-- state:derived hash:b180a0ec03a6 -->

- specs: total=445 draft=36 in-progress=18 blocked=0 done=390 superseded=0 abandoned=1
- roadmaps: total=22 active=3 done=11
- últimos closeouts:
  - spec:pose-retire-unused-first-party-action-exception (2026-10-21)
  - spec:pose-git-and-dispatch-snapshot-integrity (2026-10-21)
  - spec:pose-source-coverage-integrity (2026-10-18)
  - spec:pose-writer-coordination-and-atomic-publication (2026-10-17)
  - spec:pose-index-generation-integrity (2026-10-17)
  - ... e mais 385 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:3d5a4be62538 -->

- abertos: 110
- por criticidade: high=4 medium=22 low=65 sem-classificação=19
- vencidos (review < hoje): 25
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 15 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:6b7e7f699ecb -->

- ADRs: total=69
  - adr:2026-10-14-shared-domain-and-optional-worktree-runtime.md
  - adr:2026-10-14-review-time-context-and-reuse-provenance.md
  - adr:2026-10-14-release-temporal-evidence-and-attempt-lineage.md
  - adr:2026-10-09-obligations-may-originate-on-the-project.md
  - adr:2026-10-09-delegated-review-is-an-adapter.md
- knowledge: total=11 ativo=11 expirado=0

## Validação & Evidência
<!-- state:derived hash:d439b433c92f -->

- último registro: task=b07-snapshot-integrity outcome=pass (2026-10-21T04:10:38Z)
- últimos 30 dias: total=81 outcome_ok=59 outcome_outro=22
- reports revisados (.md): total=219
  - report:2026-10-20-b07-implementation.md
  - report:2026-10-21-standard-b07-snapshot-integrity.md
  - report:2026-10-20-b07-reconciled-review-r3.md
  - report:2026-10-20-b07-reconciled-review-r2.md
  - report:2026-10-20-b07-reconciled-review-r1.md

## Arquitetura
<!-- state:derived hash:f1a3ce7f9ca1 status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=76784 testes=65970 total=142754
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
