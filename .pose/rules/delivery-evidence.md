# Rule: Delivery Evidence

## When to consult

Consult this guide ao escrever ou revisar qualquer documento que **declare entrega,
conclusão ou prontidão** — relatórios de status, handoffs, summaries, READMEs de
módulo, seções de "estado atual" em specs/PROPOSTA, ou mensagens de "X completo".

## Convenções obrigatórias

- Declare entrega apenas com **evidência de gate verificável** anexada: comando + saída
  (`./pose validate`, `go test`, `tsc`, `vitest`) ou link ao report POSE correspondente.
- Use o vocabulário de estado do POSE: `draft` · `in-progress` · `done` · `blocked` ·
  `superseded` · `abandoned`. Não invente rótulos (`completed`, `100% COMPLETE`).
- Separe **implementado e verificado** de **planejado/documentado**. Documento de plano
  descreve intenção; não afirme que a intenção é realidade.
- Para `done`, referencie o report ou a evidência que cruzou o gate de saída da spec.
- Converta datas relativas em absolutas; carimbe a data da verificação.

## Blocking anti-patterns

- Declarar "100% COMPLETE"/"pronto para produção" sem `./pose validate --strict` verde
  no(s) módulo(s) afetado(s).
- Doc de entrega que contradiz outro doc do mesmo escopo (ex.: "delivery report" diz
  completo e "gaps analysis" diz incompleto) — reconcilie antes de publicar.
- Código mergeado com doc de conclusão mas sem passar pelos `check`/`validate` do POSE.
- Misturar aspiração e estado verificado no mesmo parágrafo sem marcação clara.

## Minimum checks

- `./pose check --strict` (estrutura + enum de status das specs).
- `./pose validate --strict` no(s) módulo(s) que o documento afirma entregar.
- `./pose lint-spec` quando o documento for uma spec.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, prevaleça a evidência verificável de `check` sobre a
  narrativa de progresso.
- When houver pressão por declarar conclusão sem gate, registre o estado real
  (`in-progress`/`blocked`) e o que falta para o gate fechar.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
