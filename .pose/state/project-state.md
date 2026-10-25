---
schema_version: 1
generated_at: 2026-10-25T13:32:17Z
baseline_commit: e20cf394cf2280b13140b939e8f53f08224086a4
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
<!-- state:derived hash:2d1682062387 -->

- specs: total=446 draft=25 in-progress=17 blocked=0 done=403 superseded=0 abandoned=1
- roadmaps: total=22 active=3 done=11
- últimos closeouts:
  - spec:pose-amendment-history-holds-before-close (2026-10-25)
  - spec:pose-closeout-terminal-verification (2026-10-25)
  - spec:pose-delegated-review-attempt-ledger (2026-10-25)
  - spec:pose-action-waiver-and-supersession-authority (2026-10-24)
  - spec:pose-policy-scope-validation (2026-10-24)
  - ... e mais 398 (ver `pose_list_specs status:done`)

## Follow-ups
<!-- state:derived hash:90da798edb5e -->

- abertos: 155
- por criticidade: high=4 medium=26 low=106 sem-classificação=19
- vencidos (review < hoje): 30
- cobertura: INCOMPLETA — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md; invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md; os números acima são o que pôde ser lido
  - spec:pose-package-manager-distribution (owner:@pose-maintainers review:2026-09-18)
  - spec:pose-federated-carried-forward-proof (owner:@harne8-platform review:2026-09-27)
  - spec:pose-review-root-binding-parity (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-verifier-assets-variable-fix (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-shellcheck-ci-gate (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-release-signing-rejection (owner:@pose-maintainers review:2026-10-02)
  - spec:pose-federated-spec-dependency-acceptance (owner:@harne8-platform review:2026-10-03)
  - spec:pose-spec-transfer-reconcile-terminal (owner:@harne8-platform review:2026-10-03)
  - spec:pose-governance-gate-activation (owner:@pose-maintainers review:2026-10-08)
  - spec:pose-package-channel-install-repair (owner:@pose-maintainers review:2026-10-08)
  - ... e mais 20 vencidos (ver `pose followups --open`)

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
<!-- state:derived hash:86f2065894ad -->

- último registro: task=validate-native outcome=pass (2026-10-24T18:00:55Z)
- últimos 30 dias: total=92 outcome_ok=66 outcome_outro=26
- reports revisados (.md): total=275
  - report:2026-10-25-b16-attempt-ledger.md
  - report:2026-10-24-b11-terminal-verification.md
  - report:2026-10-24-amendment-history-before-close.md
  - report:2026-10-24-b10-independent-review.md
  - report:2026-10-24-b10-implementation.md

## Arquitetura
<!-- state:derived hash:bfde0fc7ae61 status:active -->

- componentes: total=3 verificados=3 completude=99.8%
- linhas_de_codigo: producao=83994 testes=75524 total=159518
- linguagens: go
- saude_de_codigo: TODOs=1 FIXMEs=0 panics=0 stubs=0
- integracoes: contratos=64 ativos=1 gaps=63
- divida_tecnica: total=1 coberta=1 descoberta=0 (piso: relatório de dívida incompleto — 2 source(s) could not be read: invalid .pose/specs/2026-08-15-pose-instance-engine-version-tracking.md (scope unknown); invalid .pose/specs/2026-08-15-pose-knowledge-durable-reference-type.md (scope unknown); não certificado)
- ultimos_assessments: ver artefatos em .pose/assessments/ e .pose/state/

## Docs
<!-- state:derived hash:d5892e1cac69 -->

- manifest: ausente (rode `pose docs-init`)
