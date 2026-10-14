---
slug: delegated-review
status: draft
created_at: 2026-10-09
depends_on:
---

# Roadmap: Delegated review

**Outcome:** any project can hand a sealed review to another agent — Codex,
Claude Code or a runner such as Harne8's Conductor — and get an independent
review whose prompt the implementer did not write, whose run is recorded and
cannot be quietly retried away, and whose independence is evidence rather
than a reviewer prefix.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex
through a personal launcher (`secondary-agent`). It found three real defects
in three rounds, and exposed what the engine lacks; the [accepted ADR](../adr/2026-10-09-delegated-review-is-an-adapter.md)
records the contract and the maintainer's four decisions on 2026-10-09.

The ADR and the contract, brief and dispatch foundations are already accepted/done.
Continue the remaining draft contracts and the corrective deltas according to the
[implementation plan](../results/pose-implementation-plan.json); their existence
does not qualify the complete capability, adoption or remote integration.

## Milestone: contract
- after:
- target_start:
- target_due:
- specs: pose-delegated-review-contract

**Exit gate:** the ADR is accepted with the four maintainer decisions answered by
the maintainer, and every later spec's requirements are reconciled with
those answers.

## Milestone: brief
- after: contract
- target_start:
- target_due:
- specs: pose-delegated-review-brief

**Exit gate:** `pose review brief <bundle>` produces the same bytes for the
same sealed bundle, carries no implementer text outside a labelled notes
section, and a review run from it finds a seeded defect in a fixture.

## Milestone: dispatch
- after: brief
- target_start:
- target_due:
- specs: pose-delegated-review-dispatch

**Exit gate:** `pose review dispatch` runs a configured adapter on a
disposable worktree at the sealed commit, leaves the repository untouched,
and records a run with transcript and draft; a run that edits files or
exceeds its budget is recorded as failed, not as a review.

## Milestone: trust
- after: dispatch, attempt-ledger, spec:pose-action-answers-reverify-on-read, spec:pose-review-disclosure-requires-proof, spec:pose-historical-trust-and-review-reuse
- target_start:
- target_due:
- specs: pose-delegated-review-run-provenance

**Exit gate:** `different-actor` is satisfied from a run record (signed by a
pinned native issuer when signed attestations are adopted) and no longer
from the `agent:independent-` prefix in adopting instances; a rejection
stays visible, and rerunning an unchanged bundle without a reason is
refused.

## Milestone: adoption
- after: trust, adapter-containment, spec:pose-review-approval-domain-preflight
- target_start:
- target_due:
- specs: pose-delegated-review-capability, pose-delegated-review-action-fulfillment

**Exit gate:** `delegated-review` is a catalog capability, off by default,
documented with Codex and Claude Code adapter examples; a "review by another
actor" action request is fulfilled by a local adapter and by a non-local
runner (CI or Harne8 Conductor) through the same contract.

## Continuação prospectiva — lote 8

Consultar o [plano de continuidade de B16](../reports/2026-10-13-pose-batch-8-authoring.md).
Preservar o contrato aceito, specs done e R1–R4/status/baseline da spec reutilizada.
Separar ledger da proveniência de trust sem remover o gate histórico de rejeições.
A continuidade de B17/B18/B19/B21 está formalizada no lote 10 abaixo; B20 mantém seu lote 11. O recorte histórico do lote 8 não executou esses contratos.

## Milestone: attempt-ledger
- after: dispatch, spec:pose-source-coverage-integrity, spec:pose-writer-coordination-and-atomic-publication
- target_start:
- target_due:
- specs: pose-delegated-review-attempt-ledger

**Exit gate:** continuar R1–R4 existentes e reconciliar extensões E1–E8 do relatório
antes do start. B16 exige B06/B14 e dispatch: cota K inicia no máximo K execuções
sob concorrência; reserva durável precede efeito externo; setup/crash ficam recuperáveis;
retry não duplica execução. Corrupção/remoção não restaura orçamento ou aprovação,
rejeição permanece visível e reason não a apaga. Qualificar integridade/inventário,
limites de proteção contra writers externos e compatibilidade; escrita não entrega ledger.

## Continuação prospectiva — lote 10

Consultar o [lote 10](../reports/2026-10-13-pose-batch-10-authoring.md). Preservar
R1–R4/status da proveniência e R1–R7/status da capability; reconciliar textos
provisórios com o ADR já aceito antes do start, sem reabrir decisões do maintainer.

Trust possui B17; incorporar prospectivamente P1–P6: config/executor/prompt/output
vinculados ao bundle, prova atual verificada e independência por vendor OU modelo.
Assinatura de executor é declaração vinculada, não receipt independente do vendor;
assurance machine-attested não é pessoa nem aprovação semântica.

Adoption possui a spec compartilhada B19/B21 e a spec existente de B20 uma vez.
Executar fases B19 → B20 → B21: aceites intermediários de conclusões/destinos de
B19 precedem fulfillment B20; opt-in/rollback e eficácia integrada B21 vêm depois.
O relatório exige formalizar um gate executável de aceite corrente por fase antes
da execução; não alegar que esse gate já existe ou exigir capability done para
iniciar B20 enquanto a mesma capability precisa de B20 para seu encerramento.
Não promover aceite intermediário a spec done ou dispensar gates adotados. B20
não é implementado/formalizado pelo lote 10; sua continuidade conserva o lote 11.
O gate histórico de adoption permanece exigido; local/CI/Harne8 seguem o mesmo
contrato, com integração externa qualificada antes de ser anunciada.

## Milestone: adapter-containment
- after: spec:pose-git-and-dispatch-snapshot-integrity, spec:pose-cli-mcp-operation-results, dispatch
- target_start:
- target_due:
- specs: pose-delegated-adapter-containment

**Exit gate:** B18 exige policy explícita de isolamento e recursos, valida valores
antes de spawn e declara capacidades reais por plataforma. Streams/output/memória,
filhos/cleanup/retention têm limites e outcomes verificáveis; incompatibilidade
exigida recusa execução. Worktree e process group não são sandbox universal.
Qualificar fixtures Unix/Windows e evidência/retention, sem executar fornecedores
ou adotar capability nesta escrita. B18 exige B07/B12, não depende artificialmente de B16.
