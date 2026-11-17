---
slug: pose-doctor-asks-every-closeout-in-one-operation
status: in-progress
created_at: 2026-11-17
completed_at:
supersedes:
depends_on: pose-review-ledger-is-read-once-per-operation
priority: 0
components: pose-mcp
task_type: bugfix
---

# Spec: Doctor compartilha as capturas federadas ao listar revisões de fechamento

## 1. Intent

### Goal
Reduzir a repetição de parse de specs na consulta de revisões pendentes do doctor, preservando a decisão e sua auditoria.

### Business value
Este é o F1 aberto de pose-doctor-recommends-the-commit-graph e o bloqueador F5 do lote Harne8 hooks/follow/7.4.1 já em andamento. O perfil histórico doctor2 contém 210,95 s cumulativos em GetSpec/ArtifactResolver e 124,86 s em federatedOwnershipConflicts numa janela de 300 s (tempos sobrepostos). A operação local compartilhada deixada pela sessão anterior não habilitava snapshots novos no resolver dos projetos federados. O read view já reutilizava bytes, mas não evitava o parse repetido. O programa de archive está pausado e não é dependência desta correção.

### Constraints
Preserve a auditoria final, erros, autorização, binding de Files/resolver, raízes e cortes de ciclo. Não reutilize snapshots vindos de operação anterior. Não aumente o orçamento dos checks para esconder F5.

### Non-goals
Não implementar arquivo, catálogo novo, cache persistente ou cache de fechamento independente de cadeia, ciclo e operação. Não mudar policies nem fechar marcos reais para fazer o doctor passar.

## 2. Requirements

- R1: When doctor asks for closeout reviews owed on a versioned-index instance, the engine shall evaluate that query under one bounded review operation and publish no successful list when its final audit fails.
- R2: When a keyable review operation resolves repeated spec references in authorized projects, it shall parse each project's resolver catalogue at most once in that operation, using fresh spec snapshots over the operation's captured Files.
- R3: When operation inputs are stable, the shared resolver shall preserve resolution identity/status/errors and closeout-review findings, including missing, unauthorized and conflicting identities.
- R4: When a captured federated spec changes during the operation, including same-size content with restored mtime, the final audit shall refuse a current result.
- R5: When a new operation starts, or the prior Store escapes its callback, it shall discard prior snapshots; an incompatible binding shall remain refused and a non-keyable binding shall retain its conservative fallback.

- R6: When the doctor repeats a terminal dependency evaluation within its audited operation, a bounded memo may reuse it only when the current ancestor chain cannot introduce a cycle relative to every scope visited by that evaluation; cuts and errors shall not become cached approvals, and cache state shall not escape the query or cross clock, resolver or root contexts.

## 3. Technical Plan

### Approach
Concluir o experimento CloseoutReviewsOwed já existente com snapshots do resolver alocados depois de withGovernanceReadView e antes de fixar o ponteiro de binding de reviewOperationView. Aplicar também à entrada de scope operation, mantendo capturas independentes por escopo. Reusar ArtifactResolver.WithSpecSnapshots; não preservar o map antigo zerado intencionalmente pelo read view. Instrumentar somente o número de listSpecs por hook interno de teste, pois contar ReadFile no filesystem base não mede parse sobre bytes já capturados.

### Artifacts
- created: .pose/specs/2026-11-17-pose-doctor-asks-every-closeout-in-one-operation.md
- created: .pose/starts/pose-doctor-asks-every-closeout-in-one-operation.json
- modified: pose-mcp/internal/pose/closeout_review_policy.go
- modified: pose-mcp/internal/pose/review_operation_view.go
- modified: pose-mcp/internal/pose/spec.go
- created: pose-mcp/internal/pose/doctor_resolver_specs_test.go
- created: pose-mcp/internal/pose/doctor_closeout_memo.go
- created: pose-mcp/internal/pose/doctor_closeout_memo_test.go
- modified: pose-mcp/internal/pose/governance_read_view.go
- modified: .pose/indexes/validation-matrix.json

### Risks
O DAG ainda pode repetir verificações de fechamento por caminhos diferentes. Não memoizar isso só por ref: closeoutChain e ciclos fazem parte do contexto. A medição real deve distinguir melhoria de parse e conclusão dos checks F5. knowledge:adr-shared-worktree-runtime-review foi consultado como histórico; esta correção usa apenas reuso por operação já auditada e não ativa runtime/persistência.

## 4. Tasks

- [ ] Registrar início e implementar snapshots federados por operação.
- [ ] Provar custo, equivalência, frescor e controle negativo.
- [ ] Medir doctor e os checks de saúde antes de alegar F5 resolvido.
- [ ] Revisão independente Opus 5.5 e fechamento pelos gates nativos.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R4, R5, R6
- Date: 2026-11-17
- Context: o perfil mostra parse repetido mesmo com bytes capturados; a tentativa anterior só compartilhava o catálogo do Store primário.
- Decision: criar snapshots federados novos no início da operação, antes do binding, sem reaproveitar capturas de outra operação.
- Rationale: reutiliza a implementação existente de snapshot e conserva a auditoria completa sobre seus inputs.
- Consequences: nenhum contrato público novo; um resultado rápido só conta como correção F5 após medição dos checks completos. O perfil posterior à correção do parse ainda exige reuso conservador dos fechamentos: R6 registra esse ajuste do F5, condicionado à mesma operação e à auditoria final.

### Decision D2
- Basis: R5, R6
- Date: 2026-11-17
- Context: reuse de aprovação exige o mesmo instante local de avaliação; o fallback deve manter a semântica anterior.
- Decision: fixar clock apenas na consulta elegível ao memo; preservar clock explícito ou live do caller no fallback. Não reter positivos dependentes de prova canonicamente remota, nem resultados falsos/erros/cortes.
- Rationale: é uma otimização diagnóstica limitada à captura, sem ampliação de autoridade ou expiração e sem reuso entre queries.
- Consequences: contextos remotos permanecem conservadores; o ganho real foi medido sem retirar essa proteção.

## 6. Validation

### Strategy
Plano definido antes de aplicar o candidato ao checkout de entrega. O protótipo em clone descartável já demonstrou a hipótese; suas medições não substituem validação desta entrega.

| Cenário | Comando | Evidência esperada |
|---|---|---|
| Custo e frescor por operação | `go test ./internal/pose -run '^TestDoctorResolverCatalogue' -count=1 -v` | Uma listagem por projeto/operação; segunda operação vê mudança; mudança durante operação recusada |
| Binding, escopos e semântica de doctor | `go test ./internal/pose ./internal/cli -run 'ReviewOperationView\|ReviewScopeOperation\|CloseoutReviewPolicy\|Doctor' -count=1` | Gates existentes preservados |
| Controle negativo obrigatório | overlay somente de review_operation_view.go sem inicialização de snapshots | Teste de custo reprova por listagem repetida; checkout não é revertido |
| Módulo completo obrigatório | `pose validate --strict --module pose-mcp --jobs 1` | Resultado estruturado, sem promover timeout a aprovação |
| F5 real obrigatório | doctor e checks instance-health-composite/abm-pilot-fixture em clones no pin correto | Vereditos concluídos dentro do orçamento; se não concluírem, F5 continua aberto |

### Execution log
- Diagnóstico integrado: `pose doctor --json` em clone Harne8 38360e49 com fonte no pin 1d235ba, exit 0 em 227,467 s, sem falha de listagem de owing; avisos diagnósticos preservados. Esta medição antecede apenas o ajuste do relógio no fallback inelegível, sem alterar o ramo CLI memoizável medido. Evidência: `.pose/reports/doctor-closeout-operation/`.
- Testes focados com `-race`: pass (69,618 s), seguidos pelos testes de relógio de fallback e corte de spec remota (pass em 1,597 s). Os três overlays negativos reprovam por assertiva, não por build.
- O protótipo só com snapshots excedeu 600 s; o perfil tardio identificou a verificação repetida de predecessores. A proteção remota conservadora permanece e agora tem controle negativo.

### Known gaps
A primeira medição do protótipo ainda excedeu 600 s sem veredito; F5 não está resolvido. Três testes existentes expuseram que o wrapper anterior transformava legacy-unversioned em erro da operação; o caminho advisory legado deve permanecer como antes, sem simular geração current.
Ainda faltam os checks de saúde compostos, a validação completa e a revisão governada; F5 não está fechado. O diagnóstico independente é task de investigação, não aprovação ou atestação. O check global do pose-dist já tem 19 erros de atribuição antigos, preservados como baseline e não corrigidos por este escopo.

## 7. Final Report

### Delivered scope
Não concluída.

### Follow-ups
Sem follow-up novo; o F5 existente permanece aberto até os checks completos passarem.
