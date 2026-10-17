---
slug: pose-state-consistency-and-recovery
status: draft
created_at: 2026-10-13
depends_on:
---

# Roadmap: Consistência de estado e recuperação verificável

**Resultado:** preservar verdade de leitura, snapshot, escrita e recuperação,
sem converter unknown em autorização ou geração parcial em complete.
Consultar a [matriz](../results/pose-implementation-plan.json) e a
[análise de consistência](../reports/2026-10-12-pose-front-09-consistency-recovery.md).

Este responsável foi materializado no lote 2 para B07, antecipando somente o
arquivo previsto no lote 7. Os lotes 4 e 7 acrescentam B12/B13 e B14/B15; B34/B36 conservam seu lote.
Os sete members estão materializados em draft após o lote 9; existência não qualifica as sete unidades ou permite encerramento.

## Milestone: git-dispatch-snapshot
- after: spec:pose-recoverable-closeout-plan, spec:pose-delegated-review-dispatch
- target_start:
- target_due:
- specs: pose-git-and-dispatch-snapshot-integrity

**Exit gate:** B07 recusa clean/complete com Git desconhecido; verifica bytes e
inputs pertinentes antes/depois do dispatch, paths e outputs autorizados;
bundle superado conserva história sem efeito no scope vigente.

## Continuação prospectiva — lote 4

Consultar o [lote 4](../reports/2026-10-13-pose-batch-4-authoring.md); executar
somente quando o escopo funcional correspondente for autorizado.

## Milestone: operation-results
- after: spec:pose-recoverable-closeout-plan
- target_start:
- target_due:
- specs: pose-cli-mcp-operation-results

**Exit gate:** B12 compartilha outcome tipado entre CLI/MCP, com um JSON por
chamada, parsing estrito e distinção entre waiting/deadline/conflict, verdict,
coverage e efeitos confirmados. B12 é raiz independente de B07.

## Milestone: journal-checkpoint-lock-recovery
- after: operation-results, spec:pose-abm-atomic-start
- target_start:
- target_due:
- specs: pose-journal-checkpoint-and-lock-recovery

**Exit gate:** B13 exige EOF/schema/seq integrais; checkpoint corrupt/unavailable
não vira absent; resume revalida inputs/progresso e falha de persistência é visível.
Owner vivo/desconhecido nunca tem lock quebrado por idade. Runtime futuro deve
consumir o contrato; este milestone não declara daemon implementado.

## Continuação prospectiva — lote 7

Consultar o [lote 7](../reports/2026-10-13-pose-batch-7-authoring.md).

## Milestone: writer-publication
- after: operation-results, journal-checkpoint-lock-recovery, spec:pose-spec-authority-transfer, spec:pose-abm-atomic-start
- target_start:
- target_due:
- specs: pose-writer-coordination-and-atomic-publication

**Exit gate:** B14 usa staging exclusivo e coordenação/CAS pertinente; sem lost
update, overwrite de imutável ou dupla autoridade. Curated e efeitos antes de
erro têm preservação/receipt e recovery verificáveis; limites de writers externos explícitos.

## Milestone: index-generation
- after: writer-publication, spec:pose-delivery-integrity-index-compaction
- target_start:
- target_due:
- specs: pose-index-generation-integrity

**Exit gate:** B15 publica conjunto completo identificado ou unavailable;
reader fixa geração e retry/crash não mistura versões. Compatibilidade dos
consumers e retenção de geração viva qualificadas antes de encerramento.

## Continuação prospectiva — lote 9

Consultar o [lote 9](../reports/2026-10-13-pose-batch-9-authoring.md).
O catálogo deste responsável está materializado; implementar/qualificar cada
unidade e seus gates continua trabalho separado. Nenhum daemon é autorizado.

## Milestone: governance-read-view
- after: git-dispatch-snapshot, writer-publication, index-generation, spec:pose-source-coverage-integrity, spec:pose-obligation-projection, spec:pose-spec-authority-transfer
- target_start:
- target_due:
- specs: pose-governance-read-view-generation

**Exit gate:** B34 identifica bytes/closure efetivamente observados e revalida
inputs relevantes, incluindo journals/subjects/policy/tests/matriz/índices/refs;
interleaving causa retry limitado ou incoherence/unknown, nunca mistura ou relabel.
Mesmo mtime, dirty preexistente, rename/checkout e perda de watcher não certificam
current. Coverage perdida afeta Complete; gate/write exige revalidação/CAS no efeito,
com limites de writers externos explícitos. Qualificar antes de reuso entre operações.

## Milestone: generation-query-cursors
- after: governance-read-view, spec:pose-obligation-projection
- target_start:
- target_due:
- specs: pose-generation-bound-query-cursors

**Exit gate:** B36 vincula cursor à geração/query/filtros/ordem/escopo e oferece
snapshot identificado ou stale/restart explícito após mutação. Limitar retenção,
bytes e page size; limit reduz trabalho quando indexável e custo de total/scan é
honesto. Não truncar gates nem promover snapshot histórico a autorização atual.
Qualificar tokens, compatibilidade, empates, mutação e paridade entre consumidores.

## Ativação e encerramento

- Manter draft durante a escrita; executar somente com escopo funcional autorizado.
- Materializar e qualificar todos os deltas obrigatórios antes de encerrar o programa.
- Preservar identidade de worktree, histórico e compatibilidade; não alegar sandbox pelo check antes/depois.
- Consumir evidência real de corrida/crash/retry quando o delta correspondente existir.
- Não encerrar este roadmap pelo aceite isolado de parte dos sete members ou por resolução de suas refs.

B12 (`pose-cli-mcp-operation-results`) atingiu closeout nativo em
2026-10-16 no PR #186. O resultado tipado, os parsers estritos, o erro MCP
inconclusivo e o journal foram qualificados por revisão different-actor
Codex/OpenAI frente a Claude/Anthropic: F1–F13 fechados, 26 contraprovas,
15 probes com controles preservados, vet, suíte completa, race e quickstart.
Validação de implementação no head 16151559: global strict 134/134 e módulo
131/131. Identidade e separação declaradas, sem assinatura ou confirmação humana.
O [parecer, Eighth round](../reports/2026-10-16-pose-b12-operation-results-review.md)
e a [evidência](../results/b12-independent-review.json) identificam os selos,
atestações e gates atuais. A matriz registra B12 delivery_complete=true;
o programa continua ativo e este roadmap conserva status draft. Aguardar merge de #186 pelo
maintainer e reavaliar DAG/readiness nativa, sem selecionar nova implementação,
policy, capability ou adoção neste closeout.

B13 (`pose-journal-checkpoint-and-lock-recovery`) atingiu closeout nativo em
2026-10-16 no PR #187. Journal integral, checkpoint por efeitos observados e
recovery de trava por dono morto foram qualificados por revisão different-actor
Codex/OpenAI frente a Claude/Anthropic: F1–F4 resolvidos, 31 contraprovas por
asserção, P01–P17 com controles preservados, vet, suíte completa, race, builds
Windows/Darwin e jornada quickstart. Validação de implementação no head
8e8b1307: global strict 135/135 e módulo 132/132. Identidade e separação
declaradas, sem assinatura ou confirmação humana das conclusões.
O [parecer, Second round](../reports/2026-10-16-pose-b13-journal-checkpoint-lock-recovery-review.md)
e a [evidência](../results/b13-independent-review.json) identificam o bundle
`rvb-bbed7a49533bf1e3`, a atestação `rva-dfa2006f7d36184d` e os gates nativos. A matriz registra B13
delivery_complete=true; o programa continua ativo e este roadmap conserva seu
status. Aguardar merge de #187 pelo maintainer e reavaliar DAG/readiness nativa,
sem selecionar implementação, policy, capability ou adoção neste closeout.

B14 (`pose-writer-coordination-and-atomic-publication`) atingiu closeout
nativo em 2026-10-16 no PR #188. Publisher comum com staging exclusivo e
precondição explícita, trava de ciclo de vida compartilhada por start, cancel,
close, transfer e backfill, close vinculado aos bytes julgados e à autorização
revalidada dentro da trava, e state refresh com resultado parcial foram
qualificados em duas rodadas de revisão de ator diferente apenas por execução:
mesmo vendor e modelo do implementador (Claude Opus 5.5, Anthropic), em sessão
separada, não revisão de outro vendor. F1–F11 resolvidos ou declarados, 25/25
mutações versionadas reproduzidas e contraprovas independentes, P01–P04 iguais
ao registro, vet, suíte completa, race e builds Windows/Darwin. Validação no
head 00f743c3: global strict 136/136 e módulo 133/133. Quatro riscos baixos
foram aceitos por finding, com dono e data de revisão. Identidade declarada,
sem assinatura.
O [parecer, Second round](../reports/2026-10-16-pose-b14-writer-coordination-review.md)
e a [evidência](../results/b14-independent-review.json) identificam o bundle
`rvb-b3ff5896947dc94f`, a atestação `rva-7a84aea174407a89` e os gates nativos. A matriz registra B14
delivery_complete=true; o programa continua ativo e este roadmap conserva seu
status. Aguardar merge de #188 pelo maintainer e reavaliar DAG/readiness nativa,
sem selecionar implementação, policy, capability ou adoção neste closeout.
