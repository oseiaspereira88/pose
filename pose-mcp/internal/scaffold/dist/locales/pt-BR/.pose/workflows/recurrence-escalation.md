# Workflow: Recurrence Escalation

## Objetivo

Ativar correção sistêmica quando houver retrabalho recorrente não coberto pelos workflows atuais.

## Precondições

- Existe registro de incidentes/retrabalho por período com classificação por domínio e causa.
- O time já avaliou os workflows existentes em `.pose/workflows/` para evitar duplicação.
- O owner da área validou a necessidade de escalar para ação de processo.

## Métrica obrigatória de recorrência

Use a métrica base abaixo para detectar retrabalho recorrente:

- **Nome:** `recurrence_rework_uncovered`
- **Definição:** total de incidentes/retrabalho repetidos no período cuja causa raiz não é coberta por workflow vigente.
- **Fórmula:** `incidentes_recorrentes_nao_cobertos / periodo`
- **Dimensões mínimas:** domínio (`frontend-react`, `backend-go`, `kubernetes`, `security`, `documentation-style`) e causa (`processo`, `contrato`, `implementacao`, `validacao`).

## Limiar de ativação

Ative este workflow quando qualquer critério abaixo for atendido no período móvel de 30 dias:

- `>= 3` incidentes recorrentes não cobertos no mesmo domínio.
- `>= 5` incidentes recorrentes não cobertos no total multi-domínio.
- Tendência de crescimento por 2 períodos consecutivos (30d vs. 30d anterior).

## Checklist de execução

1. Consolidar evidência da métrica `recurrence_rework_uncovered` com recorte de 30 dias.
2. Confirmar que o padrão não está coberto por workflow vigente e registrar o gap.
3. Criar workflow especializado em `.pose/workflows/<nome>.md` com escopo, precondições, checks e saídas.
4. Vincular o novo workflow às `rules` de domínio correspondentes no próprio arquivo e no `.pose/workflows/review.md` quando aplicável.
5. Atualizar `spec` relacionada com justificativa, critérios de aceite e riscos residuais.
6. Definir owner, janela piloto e critérios de sucesso do piloto.
7. Rodar checks determinísticos aplicáveis aos arquivos alterados.
8. Registrar decisão pós-piloto: manter, ajustar ou descartar workflow.

## Vinculação obrigatória de rules

Selecione cumulativamente as `rules` por domínio afetado:

- `.pose/rules/security.md`
- `.pose/rules/backend-go.md`
- `.pose/rules/frontend-react.md`
- `.pose/rules/kubernetes.md`
- `.pose/rules/documentation-style.md`
- `.pose/rules/knowledge-governance.md` (quando houver artefatos de conhecimento/processo)

Em conflito, aplique a alternativa mais restritiva.

## Revisão de adoção (piloto)

Execute revisão após 45 dias de piloto:

- Compare volume de recorrência pré/pós ativação por domínio.
- Validar taxa de redução mínima de 30% no domínio alvo.
- Avaliar custo operacional (tempo de execução e qualidade de evidência).
- Emitir decisão formal: `manter`, `ajustar` ou `descartar`.
- Se `ajustar`/`descartar`, abrir follow-up com owner, prazo e critério de saída.

## Saídas obrigatórias

- Evidência da métrica e do limiar de ativação atingido.
- Novo workflow especializado publicado e referenciado.
- Mapeamento explícito de `rules` aplicadas.
- Resultado da revisão de piloto com decisão final.
- Riscos residuais e plano de mitigação.
