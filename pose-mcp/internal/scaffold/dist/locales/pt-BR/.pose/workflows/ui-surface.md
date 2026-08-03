# Workflow: Entrega de UI surface

## Objetivo

Provar que uma surface humana é alcançável pelo entrypoint de produção, não
apenas implementada e testada isoladamente.

## Etapas

1. Declarar `surface:<id>` em `delivers` e `### Delivery targets`.
2. Reconciliar artifacts com `pose artifact-check`.
3. Registrar checks estruturados `reachability` e `integration` ou `e2e`.
4. Rodar `pose validate --json <result-path>` e `pose surface-check --spec <slug> --strict`.
5. Rastrear a surface com `evidence:integration` ou `evidence:e2e`.
6. Executar review independente e closeout governado.

## Gate de saída

Nenhum finding obrigatório/stale e path composto atual para o mesmo provenance digest.
