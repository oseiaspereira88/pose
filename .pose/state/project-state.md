---
schema_version: 1
generated_at: 2026-11-12T17:56:37Z
baseline_commit: a0a80270fd0d5001bffa09f8181587f30dd4a08e
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
<!-- state:derived hash:bebd25e859ac -->

- specs: total=514 draft=25 in-progress=0 blocked=0 done=487 superseded=0 abandoned=2
- roadmaps: total=23 active=3 done=17
- últimos closeouts:
  - spec:pose-delegated-output-redaction-root-cause (2026-11-12)
  - spec:pose-verdict-readers-ask-the-sealed-question (2026-11-11)
  - spec:pose-commit-gate-refuses-index-without-descriptor (2026-11-11)
  - spec:pose-review-subject-head-is-the-newest (2026-11-11)
  - spec:pose-index-input-binding-reads-each-transfer-once (2026-11-10)
  - ... e mais 482 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:d624c8a11bcf -->

- abertos: 268
- por criticidade: high=5 medium=51 low=193 sem-classificação=19
- vencidos (review < hoje): 60
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 50 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:6d92b4c709dd -->

- ADRs: total=94
  - adr:2026-11-03-governance-reads-use-bounded-operation-scoped-views.md
  - adr:2026-11-02-a-milestone-of-one-member-is-that-member.md
  - adr:2026-11-01-a-milestone-answers-to-its-own-edges.md
  - adr:2026-10-31-review-reads-raw-history-and-roots-diff-from-the-empty-tree.md
  - adr:2026-10-29-make-delegated-review-adoption-explicit-and-recoverable.md
- knowledge: total=21 ativo=21 expirado=0

## Validação & Evidência
<!-- state:derived hash:2182f66db4c3 -->

- último registro: task=validate-native outcome=pass (2026-10-24T18:00:55Z)
- últimos 30 dias: total=39 outcome_ok=27 outcome_outro=12
- reports revisados (.md): total=291
  - report:2026-11-09-pose-worktree-runtime-measurements.md
  - report:2026-11-05-pose-b34-dependency-budget-measurement.md
  - report:2026-11-05-pose-b34-read-view-capacity-calibration.md
  - report:2026-11-06-pose-b34-global-review-view-budget.md
  - report:2026-10-14-security-validation-release-api-auth.md

## Arquitetura
<!-- state:derived hash:abd31a9d1ad0 status:active -->

- componentes: total=3 verificados=3 completude=99.2%
- linhas_de_codigo: producao=115203 testes=110242 total=225445
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=1 stubs=0
- integracoes: contratos=70 ativos=6 gaps=64
- divida_tecnica: total=2 coberta=1 descoberta=1 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
