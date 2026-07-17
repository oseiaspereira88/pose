---
name: pose-recurrence-escalation
description: Use quando ./pose recurrence-check sinalizar task_slug recorrente acima do threshold — investigar causa sistêmica, propor rule/workflow novo, documentar decisão e fechar o loop. Trigger keywords - recurrence, recorrência, padrão recorrente, recurrence-escalation, escalation, sistêmico, dívida recorrente.
when_to_use: Recurrence-check (manual ou em CI) flagueou ≥1 chave acima do threshold. Use ANTES de aceitar tag "intermitente" ou silenciar o sinal, para garantir tratamento sistêmico em vez de remediação localizada.
---

# Skill: pose-recurrence-escalation

Fluxo POSE para escalonar padrões detectados pelo recurrence-check.

## Required reading

1. [`.pose/workflows/recurrence-escalation.md`](../../../.pose/workflows/recurrence-escalation.md) — protocolo de escalação.
2. [`.pose/rules/_base-recurrence.md`](../../../.pose/rules/_base-recurrence.md) — princípios de rastreabilidade.
3. Histórico JSONL da chave flagueada em `.pose/reports/history/*<task_slug>*.jsonl`.

## Steps

1. Confirmar o sinal — rodar com janela mais ampla para descartar ruído:
   ```bash
   ./pose recurrence-check --tolerant --window-days 30 --threshold 3
   ```
2. Para cada chave flagged, agregar outcomes do workflow associado:
   ```bash
   ./pose stats workflows --since-days 30
   ./pose stats tasks --since-days 30 --json
   ```
3. Investigar causa sistêmica (não localizada):
   - É o mesmo módulo? Causa raiz comum?
   - É a mesma rule violada repetidamente? Falta cobertura na rule?
   - É o workflow que não previne o padrão?
4. Propor remediação:
   - Adicionar/ajustar `.pose/rules/<dominio>.md` se a causa for ausência de regra.
   - Adicionar/atualizar `.pose/workflows/<tipo>.md` se for ausência de passo no fluxo.
   - Promover check de `optional` para `required` em [`validation-matrix.json`](../../../.pose/indexes/validation-matrix.json) se a métrica em `./pose stats` justifica (taxa de sucesso ≥ 95% em 4 semanas).
5. Registrar decisão em decision-log:
   ```bash
   ./pose new-knowledge decision-log escalation-<task-slug> --owner @<dono> --ttl-days 90
   ```
6. Atualizar spec da rule/workflow alterado e abrir PR com referência ao decision-log.

## Output requirements

- Decision-log em `.pose/knowledge/` referenciando os outcomes históricos.
- PR com mudança em rule/workflow/matrix (escolher a mais barata que resolve).
- `./pose recurrence-check --strict` esperado em SUCESSO após próximo ciclo (sinaliza que a remediação funcionou).
- Atualização de [`.pose/workflows/recurrence-escalation.md`](../../../.pose/workflows/recurrence-escalation.md) se o padrão de escalação for inédito.
