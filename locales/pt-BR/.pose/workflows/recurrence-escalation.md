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
2. Classificar falhas por causa e raiz de execução (`report_path`). Uma
   tentativa de desenvolvimento com falha seguida por PASS permanece como
   evidência imutável, mas não é incidente descoberto enquanto a causa não
   sobreviver ao workflow vigente.
3. Confirmar que o padrão não está coberto por workflow vigente e registrar o gap.
4. Criar workflow especializado em `.pose/workflows/<nome>.md` com escopo, precondições, checks e saídas.
5. Vincular o novo workflow às `rules` de domínio correspondentes no próprio arquivo e no `.pose/workflows/review.md` quando aplicável.
6. Atualizar `spec` relacionada com justificativa, critérios de aceite e riscos residuais.
7. Definir owner, janela piloto e critérios de sucesso do piloto.
8. Rodar checks determinísticos aplicáveis aos arquivos alterados.
9. Registrar decisão pós-piloto: manter, ajustar ou descartar workflow.

## Vinculação obrigatória de rules

Selecione cumulativamente as `rules` por domínio afetado:

- `.pose/rules/security.md`
- `.pose/rules/backend-go.md`
- `.pose/rules/frontend-react.md`
- `.pose/rules/kubernetes.md` (extensão `pose-rule-kubernetes`, quando instalada)
- `.pose/rules/documentation-style.md`
- `.pose/rules/knowledge-governance.md` (quando houver artefatos de conhecimento/processo)

Em conflito, aplique a alternativa mais restritiva.

## Revisão de adoção (piloto)

Registre a intervenção quando a escalação for entregue, para que a revisão seja
medida e não lembrada (spec pose-recurrence-effectiveness):

```bash
pose recurrence-effect --register --task <slug-da-tarefa> \
  --ref rule:<nome>|workflow:<nome>|spec:<slug> \
  --window-days 45 --rationale "<por que esta intervenção>" --author @<alias>
```

Execute revisão após 45 dias de piloto:

- Rode `pose recurrence-effect` — ele compara a taxa de recorrência (e a
  duração/custo registrados) antes e depois da intervenção a partir do
  histórico append-only, com avisos de qualidade de dados para amostras
  esparsas ou janelas incompletas.
- Validar taxa de redução mínima de 30% no domínio alvo.
- Avaliar custo operacional (tempo de execução e qualidade de evidência)
  (`pose report --duration-seconds/--cost-usd` alimenta a telemetria).
- Emitir decisão formal: `manter`, `ajustar` ou `descartar`.
- Se `ajustar`/`descartar`, abrir follow-up com owner, prazo e critério de saída.

## Saídas obrigatórias

- Evidência da métrica e do limiar de ativação atingido.
- Novo workflow especializado publicado e referenciado.
- Mapeamento explícito de `rules` aplicadas.
- Resultado da revisão de piloto com decisão final.
- Riscos residuais e plano de mitigação.
