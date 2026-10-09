---
slug: pose-agy-smoke-test
status: draft
created_at: 2026-08-19
completed_at:
priority: 1
components: examples
---

# Spec: pose-agy-smoke-test

> Spec de teste de ponta a ponta para validação de execução automática via agente Google Antigravity (AGY CLI) conectado pelo Harne8 Desktop.

---

## 1. Intent

### Goal
Validar o ciclo completo de orquestração de tarefas, reivindicação automática (auto-claim), injeção de contexto e execução via Antigravity AGY CLI no repositório `pose-dist`.

### Business value
Comprova que o agente AGY conectado à plataforma Harne8 via Desktop executa specs sob a governança POSE, gerando artefatos verificáveis e respeitando os gates de validação.

### Constraints
- Execução determinística sem dependências externas.
- Respeitar regras canônicas do POSE no `pose-dist`.

### Non-goals
- Modificar o núcleo do binário `pose` ou regras existentes.

---

## 2. Requirements

### Functional
- R1: O agente AGY deve criar o arquivo de verificação `examples/smoke-test-agy.txt` contendo a confirmação de execução e a data/hora ISO-8601.
- R2: O agente deve executar os checks de validação POSE e garantir que a árvore permanece limpa e íntegra.

### Non-functional
- Tempo total de execução do passo inferior a 30 segundos.

### Security
- Nenhum segredo ou credencial deve ser gravado em artefatos de teste.

### Compatibility
- Compatível com Linux, macOS e Windows.

---

## 3. Technical Plan

### Affected areas
- `examples/`

### Artifacts
`examples/smoke-test-agy.txt` was planned as this spec's artifact and was never committed; nothing is claimed until a real run produces it.

### API/contract changes
- none: feature de teste sem alteração de contratos públicos

### Data/storage changes
- none: apenas gravação de arquivo de texto no diretório examples

### Technical risks
- none

---

## 4. Tasks

### Planning
- [x] Definir escopo e criar spec de teste
- [x] Validar Definition of Ready

### Implementation
- [ ] Criar arquivo `examples/smoke-test-agy.txt` com marca de execução — adiado: requer execução real do AGY pelo Harne8 Desktop
- [ ] Executar `pose validate --fast` ou checks aplicáveis — adiado: requer execução real do AGY pelo Harne8 Desktop

### Validation
- [ ] Verificar existência e integridade do arquivo criado — adiado: requer execução real do AGY pelo Harne8 Desktop
- [x] Atualizar spec para status `done`

---

## 5. Decisions

### Decisão 1: Criação de Artefato Simples em `examples/`
- Data: 2026-08-19
- Contexto: Teste de fumaça da execução de agentes AGY sem risco de quebrar o motor do POSE.
- Decisão: Criar `examples/smoke-test-agy.txt`.
- Racional: Isola completamente o teste e permite verificação imediata de sucesso.

---

## 6. Validation

### Estratégia
Validação determinística através de checagem do arquivo gerado e execução do validador POSE.

### Checks determinísticos
- Test: `test -f examples/smoke-test-agy.txt`
- Validation: `pose validate --fast`

### Log de execução
- Data:
- Ambiente:
- Notas:

### Resumo de resultados
- Sucessos:
- Falhas:
- Avisos:

### Requirement trace
<!-- No closeout, um bullet por R-ID declarado:
- R1 [deferred-integration: xref:proj.harne8/spec:harne8-pose-open-integrations-reconciliation] <the trace previously claimed this satisfied, but `examples/smoke-test-agy.txt` was never committed; the smoke needs a real AGY agent driven from Harne8 Desktop, which the Harne8 integration follow-up owns>
- R2 [deferred-integration: xref:proj.harne8/spec:harne8-pose-open-integrations-reconciliation] <depends on the same real AGY run; no run has been recorded>
-->

### Gaps conhecidos
- none

---

## 7. Final Report

### Escopo entregue
Nenhuma execução do AGY foi registrada; o trace, que afirmava os dois requisitos satisfeitos sem o arquivo existir, foi corrigido para integração adiada com dono.

### Residual risks
- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Arquivos e módulos alterados
- `examples/smoke-test-agy.txt`

### Validação executada
- Comando:
- Resultado:

### Riscos residuais
- none

### Follow-ups
- [open] Avaliar automação periódica de smoke tests em CI. (owner:@pose-maintainers crit:low review:2027-01-09)
