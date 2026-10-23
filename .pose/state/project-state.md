---
schema_version: 1
generated_at: 2026-10-23T23:20:06Z
baseline_commit: ae54756f7e8f41f661e40326e2a05821502e88f7
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
<!-- state:derived hash:76e949009a64 -->

- specs: total=446 draft=31 in-progress=17 blocked=0 done=397 superseded=0 abandoned=1
- roadmaps: total=22 active=3 done=11
- últimos closeouts:
  - spec:pose-proof-presence-channel-parity (2026-10-23)
  - spec:pose-update-dry-run-reports-the-whole-update (2026-10-23)
  - spec:pose-distributed-instructions-parity (2026-10-22)
  - spec:pose-spec-facts-evidence-binding (2026-10-22)
  - spec:pose-action-target-and-node-scoping (2026-10-22)
  - ... e mais 392 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:6357eafaa9a8 -->

- abertos: 149
- por criticidade: high=4 medium=23 low=103 sem-classificação=19
- vencidos (review < hoje): 26
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
  - ... e mais 16 vencidos (ver `pose followups --open`)

## Capabilities
<!-- state:derived hash:e20a9224b717 -->

- assessment: presente, baseline_commit=commit:dbf5b77, assessed_at=2026-08-17 (59 dias atrás)
- mecanismos: 16, score médio=4, target médio=5, retirados=0

## Decisões & Conhecimento
<!-- state:derived hash:32d1c74baf01 -->

- ADRs: total=78
  - adr:2026-10-22-update-preview-records-the-shared-execution-in-an-operation-scoped-view.md
  - adr:2026-10-21-action-effects-select-material-destinations-independently-of-origin.md
  - adr:2026-10-20-expose-the-closeout-execution-plan-as-an-opt-in-mcp-projection.md
  - adr:2026-10-18-git-state-that-cannot-be-read-is-unknown-and-a-worktree-is-judged-by-its-content.md
  - adr:2026-10-17-index-sets-are-published-as-verified-generations.md
- knowledge: total=13 ativo=13 expirado=0

## Validação & Evidência
<!-- state:derived hash:8d7e3a64476c -->

- último registro: task=validate-native outcome=pass (2026-10-23T22:56:53Z)
- últimos 30 dias: total=108 outcome_ok=79 outcome_outro=29
- reports revisados (.md): total=265
  - report:2026-10-23-b03-implementation.md
  - report:2026-10-23-b03-independent-review.md
  - report:2026-10-23-standard-validate-native.md
  - report:2026-10-23-b29-pending-review-r1.md
  - report:2026-10-23-standard-b29-security-boundary-final-source-qualification.md

## Arquitetura
<!-- state:derived hash:147d8d5fa392 status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=81858 testes=72797 total=154655
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
