# Workflow: Bugfix

## Objective

Corrigir a causa raiz com o menor impacto possível, cobertura de regressão e security operacional.

## Preconditions

- Falha reproduzida (ou evidência objetiva do defeito) está registrada.
- Scope do bug e componentes impactados estão identificados.
- Existe hipótese de causa raiz validável.
- Existe plano de validation para prevenir regressão.

## Execution checklist

1. Reproduzir o problema e definir modo de falha observável.
2. **Consultar `.pose/knowledge/`** por incidents/handoffs anteriores no mesmo módulo ou padrão de falha; reaproveitar diagnóstico já registrado.
3. Isolar causa raiz e mapear impacto colateral.
4. Definir correção mínima segura e plano de rollback.
5. Implementar fix com alteração coesa e sem refactor paralelo.
6. Adicionar/ajustar teste de regressão quando aplicável.
7. Rodar checks determinísticos relevantes (`test`, `lint`, `typecheck`, `build`).
8. Validar que o defeito foi removido e comportamento adjacente preservado.
9. **Produzir decision-log** em `.pose/knowledge/` quando a causa raiz revelar dívida sistêmica ou trade-off com impacto futuro (`./pose new-knowledge decision-log <slug>`).
10. Registrar riscos residuais e monitoramento pós-correção.

## Required outputs

- Descrição do defeito, causa raiz e abordagem de correção.
- Evidência de regressão coberta por teste ou validation equivalente.
- Resultado dos checks executados.
- Risks residuais, plano de monitoramento e rollback (quando necessário).

## Definition of done

- Defeito não reproduz mais no cenário-alvo.
- Regressão coberta por teste/validation determinística adequada.
- Não houve alteração indevida de comportamento fora do escopo.
- Checks relevantes concluídos com sucesso.

## Execução — modo implementador

**Objective:** fix the root cause with minimal changes, without expanding scope.

- **Foco:** isolamento da causa raiz antes de qualquer fix; alteração coesa sem refactor paralelo; cobertura de regressão antes do merge; comunicação clara do trade-off entre correção mínima e prevenção sistêmica.
- **Anti-padrões:** corrigir sintoma sem investigar causa; misturar bugfix com refactor não solicitado; alterar contrato público para esconder o defeito; acumular mudanças sem checkpoint de validation.
- **Handoff:** diff cirúrgico com rationale; teste de regressão executado; risco residual e monitoramento; pontos de atenção no review (especialmente trechos próximos ao fix).
