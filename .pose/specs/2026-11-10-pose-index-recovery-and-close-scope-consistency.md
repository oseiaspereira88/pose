---
slug: pose-index-recovery-and-close-scope-consistency
status: in-progress
created_at: 2026-11-10
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:index-recovery-and-close-scope-consistency
---

# Spec: O motor aponta o remédio que funciona: índice indisponível, escopo do `close` e limite do módulo

## 1. Intent

### Goal

Três respostas do motor mandavam o operador para um comando que não resolvia o que ele acabara de tentar. Cada uma vira uma resposta consistente: o que os leitores recomendam (`pose index`) repara; o `pose close` de um escopo que não é spec diz como ele fecha; e o índice de módulos não conta contra o seu limite o registro de governança que cresce a cada fechamento.

### Business value

Achado ao investigar o aviso persistente do harne8 (113 avisos do gate, todos "review input closure is unknown: index generation does not bind input …"), 2026-11-10:

1. O harne8 tinha um commit (`e1593bde`) com `spec-graph.json` e `roadmaps.json` regenerados e o `generation.json` não; os leitores respondiam "does not match generation …; run `pose index`" e o `pose index` recusava (`index input closure could not be proved … index generation could not be pinned; nothing was written`): a operação que constrói o índice a partir das fontes tratava como desconhecida a geração que ela mesma vai substituir.
2. Mesmo corrigido isso, o `pose index` do harne8 falhava com "module discovery walk: governance read view budget exceeded": o limite de 10 000 entradas da varredura de módulos foi cruzado por 7 319 arquivos rastreados do submódulo `pose-dist` (1 652 bundles e 1 597 atestados de review, 525 specs, …) mais os 2 833 do próprio harne8; o `.pose` de cada projeto não contém módulo e cresce a cada fechamento, então uma instância grande nunca mais atualizaria seus índices. A mensagem era só o nome do orçamento.
3. `pose close milestone:… --apply` respondia "the closeout plan covers spec scopes; use `pose close milestone:…` for milestone scopes": a invocação sugerida era a mesma que o operador digitou, e a diferença (o modo de plano é de spec; os outros escopos fecham sem `--apply`) não estava dita.

### Constraints

- Os leitores continuam recusando um conjunto de índices indisponível (`pose index --verify` e `ReadIndex`): só a operação que reconstrói o conjunto a partir das fontes deixa de tratar a geração indisponível como desconhecimento.
- A mesma entrada dá a mesma geração: o reparo devolve o conjunto ao que ele seria.
- Nada de `.pose` entra no inventário de módulos; o resto da varredura, os limites e as exclusões não mudam.

### Non-goals

- Impedir um commit que deixa o conjunto de índices inconsistente (o gate já o aponta como erro no modo estrito; o reparo agora existe).
- Mudar quem fecha um milestone ou o que `pose close milestone:` faz.

## 2. Requirements

- R1: `pose close <escopo> --plan|--apply|--resume` num escopo que não é spec sai com 2 e diz que o modo é o plano de fechamento de uma spec, que um escopo desse tipo fecha com `pose close <escopo>` sem nenhum modo e que se lê com `pose closeout-check <escopo>`; a mensagem do plano (também pelo MCP) não sugere a invocação recusada.
- R2: Com a geração publicada indisponível (a cópia legada de um índice não bate com o descritor, ou o diretório da geração não existe), `pose index` publica uma geração nova a partir das fontes, `pose index --verify` passa depois, e a mesma entrada dá a mesma geração; os leitores e o `--verify` antes do reparo continuam reportando indisponível.
- R3: A varredura de módulos não percorre diretórios `.pose` (do projeto, de um submódulo ou de um projeto aninhado): um registro de governança de qualquer tamanho não conta contra o limite nem entra no inventário (nem os README de `.pose`).
- R4: Quando o limite da varredura de módulos é excedido por outra coisa, o erro diz o que foi contado (mais de N entradas ou M bytes de caminhos) e que se ignora o que é gerado ou vendorizado no `.gitignore`, e continua sendo o erro de orçamento da visão de leitura.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/cli` (o `close` e os testes) e `pose-mcp/internal/pose` (a visão de leitura do índice, a varredura de módulos e a mensagem do plano), mais os manuais.

### Artifacts

- created: .pose/specs/2026-11-10-pose-index-recovery-and-close-scope-consistency.md
- created: .pose/starts/pose-index-recovery-and-close-scope-consistency.json
- created: .pose/changelogs/unreleased/pose-index-recovery-and-close-scope-consistency.md
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- modified: pose-mcp/internal/pose/governance_read_view.go
- modified: pose-mcp/internal/pose/module_discovery.go
- created: pose-mcp/internal/cli/close_plan_nonspec_test.go
- modified: pose-mcp/internal/cli/index_generation_test.go
- created: pose-mcp/internal/pose/module_discovery_governance_record_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- governance:index-recovery-and-close-scope-consistency module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose close` de um escopo que não é spec com um modo de plano passa a sair com 2 (antes saía com 1 por um erro do plano); o repo-map deixa de listar os README de `.pose`; `pose index` repara uma geração indisponível.

## 4. Tasks

- [x] Reproduzir no harne8: o `pose index` recusando a geração indisponível, depois o limite de módulos
- [x] A operação de reconstrução não trata a geração indisponível como desconhecimento
- [x] A varredura de módulos não percorre `.pose` e o erro do limite diz o que foi contado
- [x] A recusa dos modos de plano num escopo que não é spec diz como o escopo fecha
- [x] Testes que falham sem cada correção (verificado) e manuais
- [ ] Regenerar os índices do harne8 com o motor corrigido
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: documentar a restauração manual do arquivo do índice a partir do commit anterior
- Selected option: a operação que reconstrói o conjunto a partir das fontes não lê a geração publicada, então uma geração indisponível não torna o seu fechamento desconhecido
- Rationale: os índices são derivados; o comando que as mensagens dos leitores recomendam tem de poder rodar; o publicador já substitui um descritor ilegível, só a visão de captura o impedia
- Consequences: um conjunto adulterado é substituído por um reconstruído das fontes (o `--verify` o aponta antes); a geração antiga não é preservada se o diretório dela não existe
- Falsifier: um `pose index` que recusa por causa da geração indisponível, ou uma geração diferente da que as mesmas entradas dão

### Decision D2
- Basis: R3
- Minimal option: subir o limite de 10 000 entradas
- Selected option: não percorrer `.pose`
- Rationale: o limite alcança de novo qualquer valor fixo, porque o registro cresce a cada fechamento; um diretório que nunca contém módulo não precisa ser contado
- Consequences: o repo-map perde os README de `.pose`, que são documentos da própria governança
- Falsifier: um módulo ou arquivo de implantação dentro de `.pose` que o inventário deixa de ver

## 6. Validation

### Strategy

Cada teste reproduz um defeito visto no harne8 e falha sem a correção (verificado); o reparo foi também rodado no harne8 real.

### Deterministic checks

`go test` dos testes do rastreio; a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestClosePlanModesOnANonSpecScopeNameTheRightInvocation check:index-recovery-integration
- R2 [satisfied] test:TestIndexRepairsAnUnavailableGenerationThatItsReadersTellYouToRun
- R3 [satisfied] test:TestModuleDiscoveryDoesNotWalkTheGovernanceRecord
- R4 [satisfied] test:TestModuleDiscoveryNamesItsBoundWhenItIsExceeded

### Known gaps
- O motor não impede o commit que deixa o conjunto de índices inconsistente; o gate estrito o aponta e o reparo agora funciona.
- O erro do plano pelo MCP foi reescrito (a mesma função), sem teste próprio.

---

## 7. Final Report

### Delivered scope
`pose index` repara uma geração indisponível (R2); a varredura de módulos não percorre `.pose` e diz o seu limite (R3, R4); o `close` de um escopo que não é spec diz como ele fecha (R1).

### Files and modules changed
- pose-mcp/internal/cli e pose-mcp/internal/pose, os testes e os quatro POSE.md; a matriz ganha `index-recovery-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- O repo-map de uma instância muda (sem os README de `.pose`) na primeira regeneração.
- Uma instância com mais de 10 000 entradas fora de `.pose` continua parando, agora com a mensagem dizendo como ignorar.

### Follow-ups

- [open] Fazer o gate de commit recusar, no modo estrito, um commit que muda um índice sem o descritor da geração. (owner:@pose-maintainers crit:low review:2027-02-08)
