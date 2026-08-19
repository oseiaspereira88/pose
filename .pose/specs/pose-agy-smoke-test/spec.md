---
slug: pose-agy-smoke-test
status: in-progress
created_at: 2026-08-19
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority: 1            # integer >= 0 (lower = higher priority); ordering preference, not a blocker
components: examples         # optional, inline comma-separated list: modules/components touched (e.g. mcp-server, cli) — used by pose_list_specs' `components` filter
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
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
- created: `examples/smoke-test-agy.txt`

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
- [ ] Criar arquivo `examples/smoke-test-agy.txt` com marca de execução
- [ ] Executar `pose validate --fast` ou checks aplicáveis

### Validation
- [ ] Verificar existência e integridade do arquivo criado
- [ ] Atualizar spec para status `done`

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
- R1 [satisfied] check:file-exists report:examples/smoke-test-agy.txt
- R2 [satisfied] check:pose-validate
-->

### Gaps conhecidos
- none

---

## 7. Final Report

### Escopo entregue
<!-- A preencher no closeout pelo agente executor -->

### Arquivos e módulos alterados
- `examples/smoke-test-agy.txt`

### Validação executada
- Comando:
- Resultado:

### Riscos residuais
- none

### Follow-ups
- [open] Avaliar automação periódica de smoke tests em CI.
