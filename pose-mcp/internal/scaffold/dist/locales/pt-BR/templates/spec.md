---
slug: <feature-slug>
status: draft        # draft | in-progress | done | blocked | superseded | abandoned
created_at: <created_at>
completed_at:        # carimbado na transição para status: done
supersedes:          # slug da spec substituída (quando aplicável)
depends_on:          # pré-requisitos, lista inline: outra-spec, milestone:<roadmap>/<id>, roadmap:<slug>
priority:            # inteiro >= 0 (menor = mais prioritário); preferência de ordem, não pré-requisito
---

# Spec: <feature-slug>

> Template único de spec POSE. Preencha as seções relevantes; remova as que não
> se aplicam. Mantenha a ordem: Intent → Requirements → Technical Plan →
> Tasks → Decisions → Validation → Final Report.
>
> **Ciclo de vida:** atualize `status` conforme avança (`draft` → `in-progress`
> → `done`). Ao concluir, rode o fluxo de fechamento (skill `pose-spec-closeout`):
> defina `status: done`, preencha `completed_at` e dê disposição a cada follow-up.

---

## 1. Intent

### Objetivo
<!-- O que esta feature entrega, em uma frase. -->

### Valor de negócio
<!-- Por que vale a pena agora. -->

### Restrições
<!-- Limites técnicos, prazo, compliance. -->

### Não-objetivos
<!-- O que explicitamente está fora de escopo. -->

---

## 2. Requirements

> Definition of Ready (gate de entrada): antes de `status: in-progress`, os
> requisitos funcionais devem ter **acceptance criteria com IDs estáveis**
> (`- R<N>: ...`). IDs publicados não são renumerados; critério removido é
> marcado como retirado. Verifique com `pose lint-spec <slug> --ready-check`.

### Funcionais
- R1: 

### Não-funcionais
- 

### Segurança
- 

### Compatibilidade
- 

---

## 3. Technical Plan

### Áreas afetadas
- 

### Mudanças de API/contrato
- 

### Mudanças de dados/armazenamento
- 

### Riscos técnicos
- 

---

## 4. Tasks

### Planejamento
- [ ] Confirmar intent
- [ ] Identificar módulos afetados

### Implementação
- [ ] Implementar incrementalmente

### Validação
- [ ] Executar checks obrigatórios

---

## 5. Decisions

> Seção opcional. Use quando a implementação envolver trade-offs ou alternativas.

### Decisão <N>
- Data:
- Contexto:
- Opções consideradas:
- Decisão:
- Racional:
- Consequências:

---

## 6. Validation

### Estratégia
<!-- Como a feature será validada de ponta a ponta. -->

### Checks determinísticos

#### Test
- Comando:
- Escopo:
- Esperado:

#### Lint
- Comando:
- Escopo:
- Esperado:

#### Typecheck
- Comando:
- Escopo:
- Esperado:

#### Build
- Comando:
- Escopo:
- Esperado:

#### Segurança / Contrato
- Comando:
- Escopo:
- Esperado:

### Log de execução
- Data:
- Ambiente:
- Notas:

### Resumo de resultados
- Sucessos:
- Falhas:
- Avisos:

### Gaps conhecidos
<!-- Limitações temporárias, checks bloqueados, validações postergadas. -->

---

## 7. Final Report

### Escopo entregue
<!-- O que foi implementado e o que ficou de fora intencionalmente. -->

### Arquivos e módulos alterados
- 

### Validação executada
- Comando:
- Resultado:

### Riscos residuais
- 

### Follow-ups

<!--
Cada follow-up começa com uma disposição entre colchetes. Quando a spec é
marcada `status: done`, todo follow-up DEVE ter disposição (use `[open]` para
os que ainda não foram triados — `pose followups --open` os agrega).

Disposições válidas:
  [open]                  ainda não triado (backlog vivo)
  [spawned: <slug>]       virou/alimentou uma nova spec
  [covered: <slug>]       já coberto por outra spec existente
  [duplicate: <slug>]     mesmo follow-up já triado em outra spec
  [done]                  resolvido direto, sem spec separada
  [wont-do: <motivo>]     descartado conscientemente
-->

- [open] 
